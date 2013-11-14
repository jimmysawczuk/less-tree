package main

import (
	"github.com/jimmysawczuk/worker"

	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var pathToLESS string
var pathToCssMin string
var workingDirectory string
var isVerbose bool
var maxJobs int = 10

var lessFilename *regexp.Regexp

var jobs_queue worker.Worker

type CSSJob struct {
	name        string
	cmd         *exec.Cmd
	cmd_min     *exec.Cmd
	css_out     string
	css_min_out string
}

type LESSError struct {
	indent  int
	Message string
}

func init() {
	flag.StringVar(&pathToLESS, "path", "lessc", "Path to the lessc executable")
	flag.StringVar(&pathToCssMin, "css-min", "", "Path to a CSS minifier which takes an input file and spits out minified CSS in stdout")
	flag.BoolVar(&isVerbose, "v", false, "Whether or not to show LESS errors")
	flag.IntVar(&maxJobs, "max-jobs", maxJobs, "Maximum amount of jobs to run at once")
}

func main() {
	start_time := time.Now()

	flag.Parse()

	var err error
	workingDirectory, err = os.Getwd()
	if err != nil {
		panic("Can't find the working directory")
	}

	lessFilename = regexp.MustCompile(`^([A-Za-z0-9_\-\.]+)\.less$`)

	jobs_queue = worker.NewWorker()

	args := flag.Args()
	for _, v := range args {
		compileFromRoot(v)
	}

	if isVerbose {
		fmt.Println("finished building queue")
	}

	jobs_queue.On(worker.JobFinished, func(args ...interface{}) {
		pk := args[0].(*worker.Package)
		if pk.Return == 0 {
			pk.Status = worker.Finished
		} else {
			pk.Status = worker.Errored
		}
	})

	jobs_queue.RunUntilDone()

	finish_time := time.Now()

	stats := jobs_queue.Stats()
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
		if v.IsDir() && !strings.HasPrefix(v.Name(), "_") {

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

		} else if lessFilename.MatchString(v.Name()) && !strings.HasPrefix(v.Name(), "_") {

			addFile(less_dir, css_dir, v, prefix+v.Name())

		} else {

			// If we got here, it means we're either not dealing with a LESS file or we're dealing with an underscore-prefixed file (an include).
			output := ""

			switch {
			case v.IsDir() && prefix == "":
				output = v.Name() + string(os.PathSeparator) + "*"
			case v.IsDir() && prefix != "":
				output = prefix + string(os.PathSeparator) + v.Name() + string(os.PathSeparator) + "*"
			case !v.IsDir() && prefix == "":
				output = v.Name()
			case !v.IsDir() && prefix != "":
				output = prefix + string(os.PathSeparator) + v.Name()
			}

			if isVerbose {
				fmt.Printf("skip: %s\n", output)
			}
		}
	}
}

func addFile(less_dir, css_dir *os.File, less_file os.FileInfo, log_text string) {

	var cmd_min, cmd *exec.Cmd

	// normal lessc command
	cmd = exec.Command(pathToLESS, less_dir.Name()+string(os.PathSeparator)+less_file.Name())

	// if we're using a custom minifier, we want to use that here. otherwise, just use lessc with the -x flag.
	if pathToCssMin == "" {
		cmd_min = exec.Command(pathToLESS, "-x", less_dir.Name()+string(os.PathSeparator)+less_file.Name())
	} else {
		cmd_min = exec.Command(pathToCssMin, css_dir.Name()+string(os.PathSeparator)+strings.Replace(less_file.Name(), ".less", ".css", 1))
	}

	css_job := CSSJob{
		name:        log_text,
		cmd:         cmd,
		cmd_min:     cmd_min,
		css_out:     convertToCSSFilename(less_dir, css_dir, less_file, false),
		css_min_out: convertToCSSFilename(less_dir, css_dir, less_file, true),
	}

	jobs_queue.Add(css_job)
}

func convertToCSSFilename(less_dir, css_dir *os.File, less_file os.FileInfo, min bool) (css string) {
	less_filename := less_file.Name()
	css_filename := ""

	last := strings.LastIndex(less_filename, ".less")

	if last > 0 {
		if min {
			css_filename = less_filename[0:last] + ".min.css"
		} else {
			css_filename = less_filename[0:last] + ".css"
		}
	} else {
		// this shouldn't really ever happen, since we tested for it before calling this function
		css_filename = less_filename
	}

	return css_dir.Name() + string(os.PathSeparator) + css_filename
}

func (e LESSError) Error() string {
	indent_str := ""
	for i := 0; i < e.indent; i++ {
		indent_str = indent_str + " "
	}

	str := strings.Replace(fmt.Sprintf("\n%s", e.Message), "\n", "\n"+indent_str, -1)
	return str + "\n"
}

func (j CSSJob) Run(ch chan int) {

	var err error

	err = (func() error {
		result, err := j.cmd.CombinedOutput()
		if err != nil {
			return LESSError{Message: bytes.NewBuffer(result).String(), indent: 3}
		} else {
			dest_file, err := os.OpenFile(j.css_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("File write error: %s\n", err)
			} else {
				dest_file.Write(result)
				return nil
			}
		}

		return nil
	})()

	if err == nil {
		err = (func() error {
			result, err := j.cmd_min.Output()
			if err != nil {
				return LESSError{Message: bytes.NewBuffer(result).String(), indent: 3}
			} else {
				dest_file, err := os.OpenFile(j.css_min_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
				if err != nil {
					return fmt.Errorf("File write error: %s\n", err)
				} else {
					dest_file.Write(result)
					return nil
				}
			}

			return nil
		})()
	}

	exit := 0

	if err == nil {
		if isVerbose {
			fmt.Printf("ok: %s\n", j.name)
		}
	} else {
		switch err.(type) {
		case LESSError:
			fmt.Printf("err: %s\n%s", j.name, err)
			exit = 1
			break
		default:
			fmt.Printf("err: %s: %s", j.name, err)
			exit = 1
			break
		}
	}

	ch <- exit
}
