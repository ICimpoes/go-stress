# go-stress
stress testing tool

### Usage


1. Run a job for 42 seconds in 100 go routines

```go
package main

import (
	"fmt"
	"errors"
	"math/rand"
	"time"

	"github.com/icimpoes/go-stress"
)

var aJobErr = errors.New("job error")

func main() {
	results := make(map[error]int)

	jobs := []stress.Job[error]{
		{
			// the actual job which is a `func() error`
			Fn:   aJob,
			// job name (optional)
			Name: "A job",
		},
	}

	// aggregation function
	aggregate := func(r stress.Result[error]) {
		if r.JobNr%1000 == 0 {
			fmt.Println("job nr:", r.JobNr)
		}
		results[r.Data]++
	}

	maxParallel := 100

	runner := stress.New(maxParallel, jobs, aggregate)

	// stop after 42 seconds
	time.AfterFunc(42 * time.Second, runner.Stop)

	// Start runner
	// this is blocking
	runner.Start()

	fmt.Println(results)
}

func aJob() error {
	if rand.Intn(2) == 0 {
		return aJobErr
	}
	return nil
}

```

2. Run a job for 1000 times in 100 go routines
```go
package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/icimpoes/go-stress"
)

var aJobErr = errors.New("job error")

func main() {
	results := make(map[error]int)
	var totalTook time.Duration

	jobs := []stress.Job[error]{
		{
			Fn:       aJob,
			Name:     "A job",
			// limit of jobs to run
			RunTimes: 1000,
		},
	}

	aggregate := func(r stress.Result[error]) {
		if r.JobNr%1000 == 0 {
			fmt.Println("job nr:", r.JobNr)
		}
		totalTook += r.End.Sub(r.Start)
		results[r.Data]++
	}

	runner := stress.New(100, jobs, aggregate)

	runner.Start()

	fmt.Println("results:", results)
	fmt.Println("took", totalTook)
	fmt.Println("average time in ms:", float64(totalTook)/float64(time.Second))
}

func aJob() error {
	time.Sleep(10 * time.Millisecond)
	if rand.Intn(2) == 0 {
		return aJobErr
	}
	return nil
}

```

3. Run `a job` for 1000 times and `job b` for 10 seconds in 100 go routines each

```go
package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/icimpoes/go-stress"
)

var aJobErr = errors.New("job error")
var errJobB = errors.New("job error")

func main() {
	resultsA := make(map[error]int)
	resultsB := make(map[error]int)

	results := map[string]map[error]int{
		"A job": resultsA,
		"job B": resultsB,
	}

	jobs := []stress.Job[error]{
		{
			Fn:       aJob,
			Name:     "A job",
			RunTimes: 1000,
		},
		{
			Fn:   jobB,
			Name: "job B",
		},
	}

	aggregate := func(r stress.Result[error]) {
		if r.JobNr%1000 == 0 {
			fmt.Printf("job %s nr: %d\n", r.Name, r.JobNr)
		}
		results[r.Name][r.Data]++
	}

	runner := stress.New(100, jobs, aggregate)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	go func() {
		// stop after 10 seconds
		// or by SIGINT
		select {
		case <-time.After(10 * time.Second):
		case <-stop:
		}
		runner.Stop()
	}()

	runner.Start()

	fmt.Println("results:", results)
}

func aJob() error {
	time.Sleep(10 * time.Millisecond)
	if rand.Intn(2) == 0 {
		return aJobErr
	}
	return nil
}

func jobB() error {
	time.Sleep(100 * time.Millisecond)
	if rand.Intn(2) == 0 {
		return errJobB
	}
	return nil
}

```


4. Use simplified API

```go
package main

import (
	"fmt"

	"github.com/icimpoes/go-stress"
)

func main() {
	sum := 0

	ch := make(chan int)

	stop := stress.Start(10, func() int {
		return <-ch
	}, func(i int) {
		sum += i
	})

	for i := 0; i < 999999; i++ {
		ch <- 1
	}
	close(ch)
	stop()

	fmt.Println(sum)
}
```
