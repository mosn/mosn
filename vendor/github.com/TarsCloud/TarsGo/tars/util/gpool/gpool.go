package gpool

type Worker struct {
	WorkerQueue chan *Worker
	JobChannel  chan Job
	Stop        chan struct{}
}

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

type Job func()

type Pool struct {
	JobQueue    chan Job
	WorkerQueue chan *Worker
	stop        chan struct{}
}

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

func (p *Pool) Release() {
	p.stop <- struct{}{}
	<-p.stop
}
