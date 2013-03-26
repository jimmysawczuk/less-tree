package main

import (
	"time"
)

type Job interface {
	Run(chan int)
}

type Worker struct {
	jobs    []*Job
	started bool

	max_jobs     int
	running_jobs int
	total_jobs   int
}

func NewWorker() Worker {
	w := Worker{
		started:      false,
		max_jobs:     maxJobs,
		running_jobs: 0,
		total_jobs:   0,
	}

	return w
}

func (w *Worker) Add(j Job) {
	w.jobs = append(w.jobs, &j)
	w.total_jobs++
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

	if return_ch != nil {
		return_ch <- w.running_jobs
	}
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
		go (*job).Run(job_ch)
	} else {
		ch <- false
	}

	<-job_ch
	w.running_jobs--
}

func (w Worker) Total() int {
	return w.total_jobs
}
