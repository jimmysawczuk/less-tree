package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var pathToLESS string
var workingDirectory string

type Job struct {
	name        string
	cmd         *exec.Cmd
	cmd_min     *exec.Cmd
	css_out     string
	css_min_out string
}

type Worker struct {
	jobs    []*Job
	started bool

	max_jobs     int
	running_jobs int
}

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
	flag.Parse()

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

	compileDirectory("", less_dir, css_dir)
}

func compileDirectory(prefix string, less_dir, css_dir *os.File) {
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

			compileDirectory(v.Name()+"/", less_deeper, css_deeper)

		} else if v.Name()[0:1] != "_" {
			compileFile(less_dir, css_dir, v, prefix+v.Name())
		}
	}
}

func compileFile(less_dir, css_dir *os.File, less_file os.FileInfo, log_text string) {

	jobs_queue.Add(Job{
		name:        log_text,
		cmd:         exec.Command(pathToLESS, less_dir.Name()+"/"+less_file.Name()),
		cmd_min:     exec.Command(pathToLESS, "-x", less_dir.Name()+"/"+less_file.Name()),
		css_out:     css_dir.Name() + "/" + strings.Replace(less_file.Name(), ".less", ".css", 1),
		css_min_out: css_dir.Name() + "/" + strings.Replace(less_file.Name(), ".less", ".min.css", 1),
	})

}

func NewWorker() Worker {
	w := Worker{
		started:      false,
		max_jobs:     3,
		running_jobs: 0,
	}

	return w
}

func (w *Worker) Add(j Job) {
	w.jobs = append(w.jobs, &j)
}

func (w *Worker) Start(return_ch chan int) {
	if !w.started {
		w.started = true

		for len(w.jobs) > 0 {
			ch := make(chan bool)
			go w.runNextJob(ch)

			result := <-ch

			if !result {
				time.Sleep(100 * time.Millisecond)
			}
		}

		w.started = false
	}

	return_ch <- w.running_jobs
}

func (w *Worker) getNextJob() *Job {
	j := w.jobs[0]
	w.jobs = w.jobs[1:len(w.jobs)]
	return j
}

func (w *Worker) runNextJob(ch chan bool) {
	var job_ch chan int

	if w.running_jobs < w.max_jobs {
		ch <- true

		job := w.getNextJob()
		job_ch = make(chan int)
		w.running_jobs++
		go job.Run(job_ch)
	} else {
		ch <- false
	}

	<-job_ch
	w.running_jobs--
}

func (j *Job) Run(ch chan int) {

	log.Printf("Compiling %s", j.name)

	(func() {
		result, err := j.cmd.Output()
		if err != nil {
			fmt.Errorf("Command error: %s\n", err)
		} else {
			dest_file, err := os.OpenFile(j.css_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
			if err != nil {
				fmt.Errorf("File output error: %s\n", err)
			} else {
				dest_file.Write(result)
			}
		}
	})()

	(func() {
		result, err := j.cmd_min.Output()
		if err != nil {
			fmt.Errorf("Command error: %s\n", err)
		} else {
			dest_file, err := os.OpenFile(j.css_min_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
			if err != nil {
				fmt.Errorf("File output error: %s\n", err)
			} else {
				dest_file.Write(result)
			}
		}
	})()

	ch <- 1

}
