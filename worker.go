package main

import (
	"time"
)

type Job interface {
	Run(chan int)
}

type Worker struct {
	jobs Queue

	max_jobs int

	started Switch

	running_jobs Counter
	total_jobs   Counter
	success_jobs Counter
	errored_jobs Counter
}

func NewWorker() Worker {
	w := Worker{
		max_jobs: maxJobs,
	}

	return w
}

func (w *Worker) Add(j Job) {
	w.jobs.Add(j)
	w.total_jobs.AddOne()
	w.Start(nil)
}

func (w *Worker) Start(return_ch chan int) {
	if !w.started.On() && w.jobs.Len() > 0 {
		w.started.Set(true)

		for w.jobs.Len() > 0 {
			ch := make(chan int)
			go w.runNextJob(ch)
			result := <-ch

			if result == 1 {
				// this means the worker didn't accept the job because it's already running the max. We'll wait a few ms and try again.
				time.Sleep(10 * time.Millisecond)
			} else if result == 2 {
				// this should mean the worker is out of jobs, we're done
				break
			}
		}

		w.started.Set(false)
	}

	if return_ch != nil {
		return_ch <- w.running_jobs.Val()
	}
}

func (w *Worker) RunningJobs() int {
	return w.running_jobs.Val()
}

func (w *Worker) Started() bool {
	return w.started.On()
}

func (w *Worker) getNextJob() Job {
	j := w.jobs.Top()

	return *j
}

func (w *Worker) runNextJob(ch chan int) {

	running_jobs := w.RunningJobs()
	all_jobs := w.jobs.Len()

	if running_jobs < w.max_jobs && all_jobs > 0 {

		ch <- 0

		job_ch := make(chan int)
		job := w.getNextJob()

		w.running_jobs.AddOne()
		go job.Run(job_ch)
		result := <-job_ch
		w.running_jobs.SubOne()

		if result != 0 {
			w.errored_jobs.AddOne()
		} else {
			w.success_jobs.AddOne()
		}

	} else if all_jobs > 0 {

		ch <- 1

	} else if running_jobs < w.max_jobs {

		ch <- 2

	}
}
