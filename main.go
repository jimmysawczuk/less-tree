package main

import (
	//	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var pathToLESS string
var pathToCssMin string
var workingDirectory string
var isVerbose bool

var lessFilename *regexp.Regexp

var jobs_queue Worker

func main() {
	var err error
	workingDirectory, err = os.Getwd()
	if err != nil {
		log.Fatalf("Can't find the working directory.")
		os.Exit(1)
		return
	}

	flag.StringVar(&pathToLESS, "path", "lessc", "Path to the lessc executable")
	flag.StringVar(&pathToCssMin, "css-min", "", "Path to a CSS minifier which takes an input file and spits out minified CSS in stdout")
	flag.BoolVar(&isVerbose, "v", false, "Whether or not to show LESS errors")
	flag.Parse()

	lessFilename = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9_-]+)\.less$`)

	jobs_queue = NewWorker()

	args := flag.Args()
	for _, v := range args {
		compileFromRoot(v)
	}

	running_jobs_chan := make(chan int)
	running_jobs := 1
	for running_jobs > 0 {
		go jobs_queue.Start(running_jobs_chan)
		running_jobs = <-running_jobs_chan
	}

}

func compileFromRoot(dir string) {
	var fq_dir *os.File
	var err error

	if dir[0:1] != "/" {
		fq_dir, err = os.Open(workingDirectory + "/" + dir)
	} else {
		fq_dir, err = os.Open(dir)
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	less_dir, err := os.Open(fq_dir.Name() + "/less")
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("No /less directory exists at %s", fq_dir.Name())
			return
		} else {
			log.Println(err)
			return
		}
	}

	css_dir, err := os.Open(fq_dir.Name() + "/css")
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(fq_dir.Name()+"/css", 0755)
			if err != nil {
				fmt.Println("Can't create css directory")
				return
			} else {
				css_dir, _ = os.Open(fq_dir.Name() + "/css")
			}
		} else {
			log.Println(err)
			return
		}
	}

	addDirectory("", less_dir, css_dir)
}

func addDirectory(prefix string, less_dir, css_dir *os.File) {
	files, err := less_dir.Readdir(-1)
	if err != nil {
		log.Panicf("Can't parse %s", less_dir.Name())
	}

	for _, v := range files {
		if v.IsDir() {

			less_deeper, _ := os.Open(less_dir.Name() + "/" + v.Name())
			css_deeper, err := os.Open(css_dir.Name() + "/" + v.Name())
			if err != nil {
				if os.IsNotExist(err) {
					err = os.Mkdir(css_dir.Name()+"/"+v.Name(), 0755)
					if err != nil {
						fmt.Println("Can't create css directory")
						return
					} else {
						css_deeper, _ = os.Open(css_dir.Name() + "/" + v.Name())
					}
				}
			}

			addDirectory(v.Name()+"/", less_deeper, css_deeper)

		} else if lessFilename.MatchString(v.Name()) {

			addFile(less_dir, css_dir, v, prefix+v.Name())

		}
	}
}

func addFile(less_dir, css_dir *os.File, less_file os.FileInfo, log_text string) {

	var cmd_min, cmd *exec.Cmd

	// normal lessc command
	cmd = exec.Command(pathToLESS, less_dir.Name()+"/"+less_file.Name())

	// if we're using a custom minifier, we want to use that here. otherwise, just use lessc with the -x flag.
	if pathToCssMin == "" {
		cmd_min = exec.Command(pathToLESS, "-x", convertToCSSFilename(less_dir, css_dir, less_file, false))
	} else {
		cmd_min = exec.Command(pathToCssMin, css_dir.Name()+"/"+strings.Replace(less_file.Name(), ".less", ".css", 1))
	}

	jobs_queue.Add(Job{
		name:        log_text,
		cmd:         cmd,
		cmd_min:     cmd_min,
		css_out:     convertToCSSFilename(less_dir, css_dir, less_file, false),
		css_min_out: convertToCSSFilename(less_dir, css_dir, less_file, true),
	})
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

	return css_dir.Name() + "/" + css_filename
}
