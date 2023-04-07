package concurrency

type Pool struct {
	queue chan struct{}
	jobs  int
}

func NewPool(size int) *Pool {
	if size < 0 {
		size = 0
	}
	return &Pool{
		jobs:  size,
		queue: make(chan struct{}, size),
	}
}

func (p *Pool) Enqueue(job func(params ...any), params ...any) {
	if p.jobs == 0 {
		go job(params...)
		return
	}
	p.queue <- struct{}{}
	go func() {
		defer func() {
			<-p.queue
		}()
		job(params...)
	}()
}
