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
var pathToCssMin string
var workingDirectory string
var isVerbose bool
var enableCssMin bool
var maxJobs int = 4
var force bool

var version = "1.5.0"

var lessFilename *regexp.Regexp = regexp.MustCompile(`^([A-Za-z0-9_\-\.]+)\.less$`)

var analyze_queue *worker.Worker

type LESSError struct {
	indent  int
	Message string
}

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
	worker.MaxJobs = 1

	flag.Parse()

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

	if isVerbose {
		cmd := exec.Command(pathToLessc, "-v")
		out, _ := cmd.CombinedOutput()

		fmt.Printf("less-tree v%s: %s\n", version, strings.TrimSpace(string(out)))
	}

	analyze_queue = worker.NewWorker()

	args := flag.Args()
	for _, v := range args {
		compileFromRoot(v)
	}

	if isVerbose {
		fmt.Println("finished building queue")
	}

	// jobs_queue.On(worker.JobFinished, func(args ...interface{}) {
	// 	pk := args[0].(*worker.Package)
	// 	job := pk.Job().(*CSSJob)

	// 	if job.exit_code == 0 {
	// 		pk.SetStatus(worker.Finished)
	// 	} else {
	// 		pk.SetStatus(worker.Errored)
	// 	}
	// })

	analyze_queue.RunUntilDone()

	finish_time := time.Now()

	stats := analyze_queue.Stats()
	if stats.Total > 0 {
		if isVerbose {
			fmt.Println("--------------------------------------")
		}
		fmt.Printf("Compiled %d LESS files in %s\n%d ok, %d errored (%.1f%% success rate)\n",
			stats.Total,
			finish_time.Sub(start_time).String(),
			stats.Finished,
			stats.Errored,
			float64(100*stats.Finished)/float64(stats.Total),
		)
	}

}

func compileFromRoot(dir string) {
	var fq_dir *os.File
	var err error

	if !filepath.IsAbs(dir) {
		fq_dir, err = os.Open(filepath.Clean(workingDirectory + string(os.PathSeparator) + dir))
	} else {
		fq_dir, err = os.Open(filepath.Clean(dir))
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	less_dir, err := os.Open(fq_dir.Name() + string(os.PathSeparator) + "less")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No %sless directory exists at %s\n", string(os.PathSeparator), fq_dir.Name())
			return
		} else {
			fmt.Println(err)
			return
		}
	}

	css_dir, err := os.Open(fq_dir.Name() + string(os.PathSeparator) + "css")
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(fq_dir.Name()+string(os.PathSeparator)+"css", 0755)
			if err != nil {
				fmt.Printf("Can't create css directory in %s\n", fq_dir.Name())
				return
			} else {
				css_dir, _ = os.Open(fq_dir.Name() + string(os.PathSeparator) + "css")
			}
		} else {
			fmt.Println(err)
			return
		}
	}

	addDirectory("", less_dir, css_dir)
}

func addDirectory(prefix string, less_dir, css_dir *os.File) {
	files, err := less_dir.Readdir(-1)
	if err != nil {
		fmt.Printf("Can't scan %s for files", less_dir.Name())
		return
	}

	for _, v := range files {
		if v.IsDir() {
			if strings.HasPrefix(v.Name(), "_") {
				// We're dealing with an underscore-prefixed directory.
				if isVerbose {
					fmt.Printf("skip (include): %s\n", compactFilename(v, prefix))
				}

				continue
			}

			less_deeper, _ := os.Open(less_dir.Name() + string(os.PathSeparator) + v.Name())
			css_deeper, err := os.Open(css_dir.Name() + string(os.PathSeparator) + v.Name())
			if err != nil {
				if os.IsNotExist(err) {
					err = os.Mkdir(css_dir.Name()+string(os.PathSeparator)+v.Name(), 0755)
					if err != nil {
						fmt.Println("Can't create css directory")
						return
					} else {
						css_deeper, _ = os.Open(css_dir.Name() + string(os.PathSeparator) + v.Name())
					}
				}
			}

			addDirectory(v.Name()+string(os.PathSeparator), less_deeper, css_deeper)
		}

		if !v.IsDir() && lessFilename.MatchString(v.Name()) {
			if strings.HasPrefix(v.Name(), "_") {

				// We're dealing with an underscore-prefixed file (an include).
				if isVerbose {
					fmt.Printf("skip (include): %s\n", compactFilename(v, prefix))
				}

				continue
			}

			addFile(less_dir, css_dir, v, prefix+v.Name())
		}
	}
}

func addFile(less_dir, css_dir *os.File, less_file os.FileInfo, short_name string) {
	job := NewAnalyzeJob(short_name, less_dir, less_file)
	analyze_queue.Add(job)
}

func (e LESSError) Error() string {
	indent_str := ""
	for i := 0; i < e.indent; i++ {
		indent_str = indent_str + " "
	}

	str := strings.Replace(fmt.Sprintf("\n%s", e.Message), "\n", "\n"+indent_str, -1)
	return str + "\n"
}

func compactFilename(v os.FileInfo, prefix string) string {
	output := ""

	switch {
	case v.IsDir() && prefix == "":
		output = v.Name() + string(os.PathSeparator) + "*"
	case v.IsDir() && prefix != "":
		output = prefix + v.Name() + string(os.PathSeparator) + "*"
	case !v.IsDir() && prefix == "":
		output = v.Name()
	case !v.IsDir() && prefix != "":
		output = prefix + v.Name()
	}

	return output
}

func (a *lesscArg) String() string {
	return a.in
}

func (a *lesscArg) Set(in string) error {
	args := []string{}
	err := json.Unmarshal([]byte(in), &args)

	if err != nil {
		return fmt.Errorf("error parsing lessc-args (make sure it's formatted as JSON, i.e. [\"arg1\", \"arg2\"]):", err)
	}

	a.out = args

	return nil
}
