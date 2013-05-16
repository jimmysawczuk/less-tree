package main

import (
	"sync"
)

type Counter struct {
	val  int
	lock sync.RWMutex
}

type Queue struct {
	jobs []*Job
	lock sync.RWMutex
}

type Switch struct {
	val  bool
	lock sync.RWMutex
}

func (c *Counter) Add(i int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.val = c.val + i
}

func (c *Counter) Sub(i int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.val = c.val - i
}

func (c *Counter) AddOne() {
	c.Add(1)
}

func (c *Counter) SubOne() {
	c.Sub(1)
}

func (c *Counter) Val() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	v := c.val

	return v
}

func (q *Queue) Top() *Job {
	q.lock.Lock()
	defer q.lock.Unlock()

	j := q.jobs[0]
	q.jobs = q.jobs[1:]
	return j
}

func (q *Queue) Add(j Job) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.jobs = append(q.jobs, &j)
}

func (q *Queue) Len() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	l := len(q.jobs)

	return l
}

func (s *Switch) On() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	r := s.val

	return r
}

func (s *Switch) Toggle() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.val = !s.val
}

func (s *Switch) Set(v bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.val = v
}
