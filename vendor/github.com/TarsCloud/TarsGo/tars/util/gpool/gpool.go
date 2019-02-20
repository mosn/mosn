package gpool

//Worker goroutine struct.
type Worker struct {
	WorkerQueue chan *Worker
	JobChannel  chan Job
	Stop        chan struct{}
}

//Start start gotoutine pool.
func (w *Worker) Start() {
	go func() {
		var job Job
		for {
			w.WorkerQueue <- w
			select {
			case job = <-w.JobChannel:
				job()
			case <-w.Stop:
				w.Stop <- struct{}{}
				return
			}
		}
	}()
}

func newWorker(pool chan *Worker) *Worker {
	return &Worker{
		WorkerQueue: pool,
		JobChannel:  make(chan Job),
		Stop:        make(chan struct{}),
	}
}

//Job is a function for doing jobs.
type Job func()

//Pool is goroutine pool config.
type Pool struct {
	JobQueue    chan Job
	WorkerQueue chan *Worker
	stop        chan struct{}
}

//NewPool news gotouine pool
func NewPool(numWorkers int, jobQueueLen int) *Pool {
	jobQueue := make(chan Job, jobQueueLen)
	workerQueue := make(chan *Worker, numWorkers)

	pool := &Pool{
		JobQueue:    jobQueue,
		WorkerQueue: workerQueue,
		stop:        make(chan struct{}),
	}
	pool.Start()
	return pool
}

//Start starts all workers
func (p *Pool) Start() {
	for i := 0; i < cap(p.WorkerQueue); i++ {
		worker := newWorker(p.WorkerQueue)
		worker.Start()
	}

	go p.dispatch()
}

func (p *Pool) dispatch() {
	for {
		select {
		case job := <-p.JobQueue:
			worker := <-p.WorkerQueue
			worker.JobChannel <- job
		case <-p.stop:
			for i := 0; i < cap(p.WorkerQueue); i++ {
				worker := <-p.WorkerQueue

				worker.Stop <- struct{}{}
				<-worker.Stop
			}

			p.stop <- struct{}{}
			return
		}
	}
}

//Release release all workers
func (p *Pool) Release() {
	p.stop <- struct{}{}
	<-p.stop
}
