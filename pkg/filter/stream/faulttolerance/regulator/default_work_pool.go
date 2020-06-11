/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package regulator

import (
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/pkg/utils"
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
