package main

import (
	"github.com/jimmysawczuk/worker"

	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var pathToLessc string
var lesscArgs lesscArg
var pathToCSSMin string
var workingDirectory string
var isVerbose bool
var enableCSSMin bool
var force bool
var maxJobs = 4
var version = "1.6.0"
var lessFilename = regexp.MustCompile(`^([A-Za-z0-9_\-\.]+)\.less$`)

type lesscArg struct {
	in  string
	out []string
}

func init() {
	flag.StringVar(&pathToLessc, "lessc-path", "lessc", "Path to the lessc executable")
	flag.Var(&lesscArgs, "lessc-args", "Any extra arguments/flags to pass to lessc before the paths (specified as a JSON array)")

	flag.BoolVar(&isVerbose, "v", false, "Whether or not to show LESS errors")
	flag.IntVar(&maxJobs, "max-jobs", maxJobs, "Maximum amount of jobs to run at once")
	flag.BoolVar(&force, "f", false, "If true, all CSS will be rebuilt regardless of whether or not the source LESS file(s) changed")

	flag.BoolVar(&enableCSSMin, "min", false, "Automatically minify outputted css files")
	flag.StringVar(&pathToCSSMin, "cssmin-path", "", "Path to cssmin (or an executable which takes an input file as an argument and spits out minified CSS in stdout)")

	flag.Usage = func() {
		cmd := exec.Command(pathToLessc, "-v")
		out, err := cmd.CombinedOutput()
		if err != nil {
			out = []byte("lessc not found")
		}

		fmt.Printf("less-tree version %s; %s\n", version, strings.TrimSpace(string(out)))
		fmt.Printf("Usage: less-tree [options] <dir> <another-dir>...\n")
		flag.PrintDefaults()
	}
}

func main() {
	start := time.Now()

	flag.Parse()
	worker.MaxJobs = maxJobs

	validateEnvironment()

	if isVerbose {
		cmd := exec.Command(pathToLessc, "-v")
		out, _ := cmd.CombinedOutput()

		fmt.Printf("less-tree v%s: %s\n", version, strings.TrimSpace(string(out)))
	}

	cssQueue := worker.NewWorker()
	cssQueue.On(worker.JobFinished, func(pk *worker.Package, args ...interface{}) {
		job := pk.Job().(*cssJob)

		if job.exitCode == 0 {
			pk.SetStatus(worker.Finished)
		} else {
			pk.SetStatus(worker.Errored)
		}
	})

	args := flag.Args()
	for _, v := range args {
		parseDirectory(v, cssQueue)
	}

	cssQueue.RunUntilDone()

	finish := time.Now()

	if len(args) > 0 {
		stats := cssQueue.Stats()

		successRate := float64(0)
		if stats.Total > 0 {
			successRate = float64(100*stats.Finished) / float64(stats.Total)
		}

		if isVerbose {
			fmt.Println("--------------------------------------")
		}
		fmt.Printf("Compiled %d LESS files in %s\n%d ok, %d errored (%.1f%% success rate)\n",
			stats.Total,
			finish.Sub(start).String(),
			stats.Finished,
			stats.Errored,
			successRate,
		)
	}
}

func (a *lesscArg) String() string {
	return a.in
}

func (a *lesscArg) Set(in string) error {
	args := []string{}
	err := json.Unmarshal([]byte(in), &args)

	if err != nil {
		return fmt.Errorf("error parsing lessc-args (make sure it's formatted as JSON, i.e. [\"arg1\", \"arg2\"]): %s", err)
	}

	a.out = args

	return nil
}

func validateEnvironment() {
	var err error
	workingDirectory, err = os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Can't find the working directory")
		os.Exit(1)
	}

	path, err := exec.LookPath(pathToLessc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "The lessc path provided (%s) is invalid\n", path)
		os.Exit(1)
	}

	if enableCSSMin {
		if pathToCSSMin == "" {
			fmt.Fprintf(os.Stderr, "CSS minification invoked but no path provided\n")
			os.Exit(1)
		}

		path, err := exec.LookPath(pathToCSSMin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CSS minification invoked but the path provided (%s) is invalid\n", path)
			os.Exit(1)
		}
	}
}

func parseDirectory(dir string, cssQueue *worker.Worker) {
	analyzeQueue := worker.NewWorker()
	lessFileCh := make(chan *lessFile, 100)
	errCh := make(chan error, 100)
	stopCh := make(chan bool)
	files := []*lessFile{}

	crawler, err := newDirectoryCrawler(dir, func(crawler *directoryCrawler, less_dir, css_dir *os.File, less_file os.FileInfo) {
		name, _ := filepath.Rel(crawler.rootLESS.Name(), filepath.Join(less_dir.Name(), less_file.Name()))
		job := newFindImportsJob(name, less_dir, css_dir, less_file, lessFileCh, errCh)
		analyzeQueue.Add(job)
	})
	if err != nil {
		fmt.Printf("error crawling directory %s: %s\n", dir, err)
	}

	cm := newLessTreeCache(crawler.rootCSS)
	err = cm.Load()

	go func(less_file_ch chan *lessFile, error_ch chan error, stop_ch chan bool) {
		for {
			select {
			case l := <-less_file_ch:
				files = append(files, l)

			case err := <-error_ch:
				fmt.Printf("err: %s\n", err)

			case _ = <-stop_ch:
				break
			}
		}
	}(lessFileCh, errCh, stopCh)

	crawler.Parse()

	if isVerbose {
		fmt.Println("finished building queue")
	}

	analyzeQueue.RunUntilDone()
	stopCh <- true

	for _, file := range files {
		job := newCSSJob(file.Name, file.Dir, file.CSSDir, file.File, lesscArgs.out)
		isCached := cm.Test(file)
		outputFilesExist := job.OutputFilesExist()

		if !isCached || !outputFilesExist || force {
			cssQueue.Add(job)
		}
	}

	cm.Save()
}
