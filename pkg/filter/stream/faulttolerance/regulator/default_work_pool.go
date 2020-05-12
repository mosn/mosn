package regulator

import (
	"math/rand"
	"mosn.io/pkg/utils"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type WorkGoroutine struct {
	tasks *sync.Map
}

func NewWorkGoroutine() *WorkGoroutine {
	worker := &WorkGoroutine{
		tasks: new(sync.Map),
	}
	return worker
}

func (g *WorkGoroutine) AddTask(key string, model *MeasureModel) {
	g.tasks.Store(key, model)
}

func (g *WorkGoroutine) Start() {
	utils.GoWithRecover(func() {
		tick := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-tick.C:
				g.work()
			}
		}
	}, func(r interface{}) {
		g.Start()
	})
}

func (g *WorkGoroutine) work() {
	g.tasks.Range(func(key, value interface{}) bool {
		model := value.(*MeasureModel)
		if model.IsArrivalTime() {
			model.Measure()
		}
		return true
	})
}

type DefaultWorkPool struct {
	size           int64
	index          int64
	workers        *sync.Map
	randomInstance *rand.Rand
	lock           *sync.Mutex
}

func NewDefaultWorkPool(size int64) *DefaultWorkPool {
	workPool := &DefaultWorkPool{
		size:           size,
		workers:        new(sync.Map),
		randomInstance: rand.New(rand.NewSource(time.Now().UnixNano())),
		lock:           new(sync.Mutex),
	}
	return workPool
}

func (w *DefaultWorkPool) roundRobin() string {
	index := atomic.AddInt64(&w.index, 1) % w.size
	return strconv.FormatInt(index, 10)
}

func (w *DefaultWorkPool) Schedule(model *MeasureModel) {
	index := w.roundRobin()
	if value, ok := w.workers.Load(index); ok {
		worker := value.(*WorkGoroutine)
		worker.AddTask(model.GetKey(), model)
	} else {
		worker := NewWorkGoroutine()
		worker.AddTask(model.GetKey(), model)
		if _, ok := w.workers.LoadOrStore(index, worker); !ok {
			worker.Start()
		} else {
			worker.AddTask(model.GetKey(), model)
		}
	}
}
