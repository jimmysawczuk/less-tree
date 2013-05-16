package main

import (
	"sync"
	"time"
	// "fmt"
)

type Job interface {
	Run(chan int)
}

type Worker struct {
	jobs Queue

	started    bool
	start_lock sync.RWMutex

	max_jobs     int
	running_jobs Counter

	total_jobs   Counter
	success_jobs Counter
	errored_jobs Counter
}

func NewWorker() Worker {
	w := Worker{
		started:  false,
		max_jobs: maxJobs,
	}

	return w
}

func (w *Worker) Add(j Job) {
	w.jobs.Add(j)
	w.total_jobs.AddOne()
}

func (w *Worker) Started() bool {
	w.start_lock.RLock()
	defer w.start_lock.RUnlock()

	r := w.started

	return r
}

func (w *Worker) Start(return_ch chan int) {
	if !w.Started() {
		w.start_lock.Lock()
		w.started = true
		w.start_lock.Unlock()

		for w.jobs.Len() > 0 {
			ch := make(chan bool)
			go w.runNextJob(ch)

			result := <-ch

			if !result {
				time.Sleep(1 * time.Millisecond)
			}
		}

		w.start_lock.Lock()
		w.started = false
		w.start_lock.Unlock()
	} else {
		time.Sleep(1 * time.Millisecond)
	}

	if return_ch != nil {
		return_ch <- w.running_jobs.Val()
	}
}

func (w *Worker) getNextJob() *Job {
	return w.jobs.Top()
}

func (w *Worker) runNextJob(ch chan bool) {
	var job_ch chan int

	if w.running_jobs.Val() < w.max_jobs {
		ch <- true

		job := w.getNextJob()
		job_ch = make(chan int)
		w.running_jobs.AddOne()
		go (*job).Run(job_ch)
	} else {
		ch <- false
	}

	result := <-job_ch
	w.running_jobs.SubOne()

	if result != 0 {
		w.errored_jobs.AddOne()
	} else {
		w.success_jobs.AddOne()
	}
}
