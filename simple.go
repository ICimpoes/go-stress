package stress

type JobFn[T any] func() T

func Start[T any](p int, job JobFn[T], agg func(T)) func() {
	r := New(p, []Job[T]{{Fn: job}}, func(r Result[T]) {
		agg(r.Data)
	})

	done := make(chan bool)
	go func() {
		r.Start()
		close(done)
	}()

	return func() {
		r.Stop()
		<-done
	}
}
