package stress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Job[T any] struct {
	Fn          func() T
	Name        string
	RunTimes    int
	MaxParallel int
}

type Result[T any] struct {
	Data  T
	Name  string
	JobNr int
	Start time.Time
	End   time.Time
}

type runner[T any] struct {
	wg      sync.WaitGroup
	done    chan struct{}
	results chan Result[T]

	logWriter io.Writer

	maxParallel int
	jobs        []Job[T]
	aggregate   func(Result[T])
}

func New[T any](maxParallel int, jobs []Job[T], aggregate func(Result[T])) *runner[T] {
	return &runner[T]{
		done:        make(chan struct{}),
		results:     make(chan Result[T], maxParallel),
		maxParallel: maxParallel,
		jobs:        jobs,
		aggregate:   aggregate,
		logWriter:   os.Stdout,
	}
}

func (r *runner[T]) SetLogWriter(w io.Writer) {
	r.logWriter = w
}

func (r *runner[T]) Stop() {
	close(r.done)
}

func (r *runner[T]) Start() {
	for _, job := range r.jobs {
		r.wg.Add(1)
		go func(job Job[T]) {
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

func (r *runner[T]) listenResults() {
	for res := range r.results {
		r.aggregate(res)
	}
}

func (r *runner[T]) runJob(job Job[T]) {
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
				res := job.Fn()
				end := time.Now()
				<-ch
				r.results <- Result[T]{Name: job.Name, Data: res, Start: start, End: end, JobNr: i}
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
