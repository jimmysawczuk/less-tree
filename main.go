package main

import (
	"github.com/jimmysawczuk/worker"
	"github.com/pkg/errors"

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
var version = "1.7.0"
var lessFilename = regexp.MustCompile(`^([A-Za-z0-9_\-\.]+)\.less$`)

type lesscArg struct {
	in  string
	out []string
}

func init() {
	flag.StringVar(&pathToLessc, "lessc-path", "", "Path to the lessc executable")
	flag.Var(&lesscArgs, "lessc-args", "Any extra arguments/flags to pass to lessc before the paths (specified as a JSON array)")

	flag.BoolVar(&isVerbose, "v", false, "Whether or not to show LESS errors")
	flag.IntVar(&maxJobs, "max-jobs", maxJobs, "Maximum amount of jobs to run at once")
	flag.BoolVar(&force, "f", false, "If true, all CSS will be rebuilt regardless of whether or not the source LESS file(s) changed")

	flag.BoolVar(&enableCSSMin, "min", false, "Automatically minify outputted css files")
	flag.StringVar(&pathToCSSMin, "cssmin-path", "", "Path to cssmin (or an executable which takes an input file as an argument and spits out minified CSS in stdout)")

	flag.Usage = func() {
		versions()
		fmt.Printf("Usage: less-tree [options] <dir> <another-dir>...\n")
		flag.PrintDefaults()
	}
}

func versions() {
	cmd := exec.Command(pathToLessc, "-v")
	lesscVersion, err := cmd.CombinedOutput()
	if err != nil {
		lesscVersion = []byte("lessc not found!")
	}

	fmt.Printf("less-tree v%s\n", version)
	fmt.Printf(" - lessc (%s): %s\n", pathToLessc, strings.TrimSpace(string(lesscVersion)))
	fmt.Printf(" - cssmin (%s): enabled: %t\n", pathToCSSMin, enableCSSMin)
	fmt.Printf("\n")
}

func main() {
	start := time.Now()

	flag.Parse()
	worker.MaxJobs = maxJobs

	err := validateEnvironment()
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "less-tree"))

		// these are all path errors, so exit(1) should be okay/meaningful.
		os.Exit(1)
		return
	}

	if isVerbose {
		versions()
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
		return errors.Wrap(err, "error parsing lessc-args (make sure it's formatted as JSON, i.e. [\"arg1\", \"arg2\"])")
	}

	a.out = args

	return nil
}

func validateEnvironment() error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.New("can't find the working directory")
	}

	workingDirectory = wd

	// if the path to lessc is explicitly provided and we can't find it, that's a big problem
	if pathToLessc != "" {
		path, err := exec.LookPath(pathToLessc)
		if err != nil {
			return errors.Errorf("the lessc path provided (%s) is invalid", pathToLessc)
		}
		pathToLessc = path
	} else {
		paths := []string{
			"./node_modules/.bin/lessc",
			"lessc",
		}
		lesscFound := false

		for _, path := range paths {
			p, err := exec.LookPath(path)
			if err == nil {
				lesscFound = true
				pathToLessc = p
				break
			}
		}

		if !lesscFound {
			return errors.New("couldn't find lessc executable from the inferred paths: " + strings.Join(paths, "; "))
		}
	}

	// Only validate the cssmin executable if we're actually trying to use it
	if enableCSSMin {

		// if the path to cssmin is explicitly provided and we can't find it, that's a big problem
		if pathToCSSMin != "" {
			path, err := exec.LookPath(pathToCSSMin)
			if err != nil {
				return errors.Errorf("the cssmin path provided (%s) is invalid", pathToCSSMin)
			}
			pathToCSSMin = path
		} else {
			paths := []string{
				"./node_modules/.bin/cssmin",
				"cssmin",
			}
			cssminFound := false

			for _, path := range paths {
				p, err := exec.LookPath(path)
				if err == nil {
					cssminFound = true
					pathToCSSMin = p
					break
				}
			}

			if !cssminFound {
				return errors.New("couldn't find lessc executable from the inferred paths: " + strings.Join(paths, "; "))
			}
		}
	}

	return nil
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
