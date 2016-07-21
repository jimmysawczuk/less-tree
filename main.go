package main

import (
	"github.com/jimmysawczuk/less-tree/less"
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
var pathToCssMin string
var workingDirectory string
var isVerbose bool
var enableCssMin bool
var maxJobs int = 4
var force bool

var version = "1.5.4"

var lessFilename *regexp.Regexp = regexp.MustCompile(`^([A-Za-z0-9_\-\.]+)\.less$`)

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

	flag.BoolVar(&enableCssMin, "min", false, "Automatically minify outputted css files")
	flag.StringVar(&pathToCssMin, "cssmin-path", "", "Path to cssmin (or an executable which takes an input file as an argument and spits out minified CSS in stdout)")

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
	start_time := time.Now()

	flag.Parse()
	worker.MaxJobs = maxJobs

	validateEnvironment()

	if isVerbose {
		cmd := exec.Command(pathToLessc, "-v")
		out, _ := cmd.CombinedOutput()

		fmt.Printf("less-tree v%s: %s\n", version, strings.TrimSpace(string(out)))
	}

	css_queue := worker.NewWorker()
	css_queue.On(worker.JobFinished, func(pk *worker.Package, args ...interface{}) {
		job := pk.Job().(*CSSJob)

		if job.exit_code == 0 {
			pk.SetStatus(worker.Finished)
		} else {
			pk.SetStatus(worker.Errored)
		}
	})

	args := flag.Args()
	for _, v := range args {
		analyze_queue := worker.NewWorker()
		less_file_ch := make(chan *less.LESSFile, 100)
		error_ch := make(chan error, 100)
		stop_ch := make(chan bool)

		crawler, err := NewDirectoryCrawler(v, func(crawler *DirectoryCrawler, less_dir, css_dir *os.File, less_file os.FileInfo) {
			short_name, _ := filepath.Rel(crawler.rootLESS.Name(), filepath.Join(less_dir.Name(), less_file.Name()))
			job := NewFindImportsJob(short_name, less_dir, css_dir, less_file, less_file_ch, error_ch)
			analyze_queue.Add(job)
		})
		if err != nil {
			fmt.Printf("error crawling directory %s: %s\n", v, err)
		}

		cm := NewLessTreeCache(crawler.rootCSS)
		err = cm.Load()

		files := make([]*less.LESSFile, 0)

		go func(less_file_ch chan *less.LESSFile, error_ch chan error, stop_ch chan bool) {
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
		}(less_file_ch, error_ch, stop_ch)

		crawler.Parse()

		if isVerbose {
			fmt.Println("finished building queue")
		}

		analyze_queue.RunUntilDone()
		stop_ch <- true

		for _, file := range files {
			job := NewCSSJob(file.Name, file.Dir, file.CSSDir, file.File, lesscArgs.out)
			is_cached := cm.Test(file)
			output_files_exist := job.OutputFilesExist()

			if !is_cached || !output_files_exist || force {
				css_queue.Add(job)
			}
		}

		cm.Save()
	}

	css_queue.RunUntilDone()

	finish_time := time.Now()

	if len(args) > 0 {
		stats := css_queue.Stats()

		success_rate := float64(0)
		if stats.Total > 0 {
			success_rate = float64(100*stats.Finished) / float64(stats.Total)
		}

		if isVerbose {
			fmt.Println("--------------------------------------")
		}
		fmt.Printf("Compiled %d LESS files in %s\n%d ok, %d errored (%.1f%% success rate)\n",
			stats.Total,
			finish_time.Sub(start_time).String(),
			stats.Finished,
			stats.Errored,
			success_rate,
		)
	}
}

func (e LESSError) Error() string {
	indent_str := ""
	for i := 0; i < e.indent; i++ {
		indent_str = indent_str + " "
	}

	str := strings.Replace(fmt.Sprintf("\n%s", e.Message), "\n", "\n"+indent_str, -1)
	return str + "\n"
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

	if enableCssMin {
		if pathToCssMin == "" {
			fmt.Fprintf(os.Stderr, "CSS minification invoked but no path provided\n")
			os.Exit(1)
		}

		path, err := exec.LookPath(pathToCssMin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CSS minification invoked but the path provided (%s) is invalid\n", path)
			os.Exit(1)
		}
	}
}
