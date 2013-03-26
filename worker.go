package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

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

	compile_error := false

	(func() {
		result, err := j.cmd.CombinedOutput()
		if err != nil {
			if isVerbose {
				fmt.Println(bytes.NewBuffer(result).String())
			}

			compile_error = true
		} else {
			dest_file, err := os.OpenFile(j.css_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
			if err != nil {
				log.Println(fmt.Errorf("File output error: %s\n", err))
			} else {
				dest_file.Write(result)
			}
		}
	})()

	if !compile_error {
		(func() {
			result, err := j.cmd_min.Output()
			if err != nil {
				if isVerbose {
					fmt.Println(bytes.NewBuffer(result).String())
				}

				compile_error = true
			} else {
				dest_file, err := os.OpenFile(j.css_min_out, os.O_RDWR+os.O_TRUNC+os.O_CREATE, 0644)
				if err != nil {
					log.Println(fmt.Errorf("File output error: %s\n", err))
				} else {
					dest_file.Write(result)
				}
			}
		})()
	}

	if !compile_error {
		fmt.Printf("SUCCESS: %s compiled\n", j.name)
	} else {
		if !isVerbose {
			fmt.Printf("ERROR: %s NOT compiled, run with -v for errors\n", j.name)
		} else {
			fmt.Printf("ERROR: %s NOT compiled\n", j.name)
		}

	}

	ch <- 1
}
