package stress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Job struct {
	Fn          func() error
	Name        string
	RunTimes    int
	MaxParallel int
}

type Result struct {
	Error error
	Name  string
	JobNr int
	Took  time.Duration
}

type runner struct {
	wg      sync.WaitGroup
	done    chan struct{}
	results chan Result

	logWriter io.Writer

	maxParallel int
	jobs        []Job
	aggregate   func(Result)
}

func New(maxParallel int, jobs []Job, aggregate func(Result)) *runner {
	return &runner{
		done:        make(chan struct{}),
		results:     make(chan Result, maxParallel),
		maxParallel: maxParallel,
		jobs:        jobs,
		aggregate:   aggregate,
		logWriter:   os.Stdout,
	}
}

func (r *runner) SetLogWriter(w io.Writer) {
	r.logWriter = w
}

func (r *runner) Stop() {
	close(r.done)
}

func (r *runner) Start() {
	for _, job := range r.jobs {
		r.wg.Add(1)
		go func(job Job) {
			r.runJob(job)
			r.wg.Done()
		}(job)
	}
	go func() {
		r.wg.Wait()
		fmt.Fprintf(r.logWriter, "all jobs finished\n")
		close(r.results)
	}()
	r.listenResults()
}

func (r *runner) listenResults() {
	for res := range r.results {
		r.aggregate(res)
	}
}

func (r *runner) runJob(job Job) {
	maxParallel := job.MaxParallel
	if maxParallel <= 0 {
		maxParallel = r.maxParallel
	}
	ch := make(chan struct{}, maxParallel)
	wg := sync.WaitGroup{}
	forever := job.RunTimes <= 0
loop:
	for i := 1; forever || job.RunTimes-i >= 0; i++ {
		select {
		case ch <- struct{}{}:
			wg.Add(1)
			go func(i int) {
				start := time.Now()
				err := job.Fn()
				took := time.Since(start)
				<-ch
				r.results <- Result{Name: job.Name, Error: err, Took: took, JobNr: i}
				wg.Done()
			}(i)
		case <-r.done:
			fmt.Fprintf(r.logWriter, "%s, received done in '%s' job, waiting to finish\n", time.Now().String(), job.Name)
			break loop
		}
	}
	wg.Wait()
	fmt.Fprintf(r.logWriter, "%s, '%s' job is done\n", time.Now(), job.Name)
	return
}
