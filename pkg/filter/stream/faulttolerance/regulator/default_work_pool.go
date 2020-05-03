package regulator

import (
	"math/rand"
	"mosn.io/pkg/utils"
	"strconv"
	"strings"
	"sync"
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

func (g *WorkGoroutine) getApp(key string) string {
	if index := strings.Index(key, "@"); index != -1 {
		if app := key[:index]; app != "" {
			return app
		}
	}
	return key
}

type DefaultWorkPool struct {
	size           int64
	workSize       int64
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

func (w *DefaultWorkPool) random() string {
	w.lock.Lock()
	defer w.lock.Unlock()
	return strconv.FormatInt(w.randomInstance.Int63n(w.size), 10)
}

func (w *DefaultWorkPool) Schedule(model *MeasureModel) {
	index := w.random()
	if value, ok := w.workers.Load(index); ok {
		worker := value.(*WorkGoroutine)
		worker.AddTask(model.GetKey(), model)
	} else {
		worker := NewWorkGoroutine()
		worker.AddTask(model.GetKey(), model)
		if _, ok := w.workers.LoadOrStore(index, worker); !ok {
			worker.Start()
			w.workSize++
		} else {
			worker.AddTask(model.GetKey(), model)
		}
	}
}
