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

package gxsync

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
)

import (
	gxruntime "github.com/dubbogo/gost/runtime"
)

const (
	defaultTaskQNumber = 10
	defaultTaskQLen    = 128
)

/////////////////////////////////////////
// Task Pool Options
/////////////////////////////////////////

type TaskPoolOptions struct {
	tQLen      int // task queue length. buffer size per queue
	tQNumber   int // task queue number. number of queue
	tQPoolSize int // task pool size. number of workers
}

func (o *TaskPoolOptions) validate() {
	if o.tQPoolSize < 1 {
		panic(fmt.Sprintf("illegal pool size %d", o.tQPoolSize))
	}

	if o.tQLen < 1 {
		o.tQLen = defaultTaskQLen
	}

	if o.tQNumber < 1 {
		o.tQNumber = defaultTaskQNumber
	}

	if o.tQNumber > o.tQPoolSize {
		o.tQNumber = o.tQPoolSize
	}
}

type TaskPoolOption func(*TaskPoolOptions)

// @size is the task queue pool size
func WithTaskPoolTaskPoolSize(size int) TaskPoolOption {
	return func(o *TaskPoolOptions) {
		o.tQPoolSize = size
	}
}

// @length is the task queue length
func WithTaskPoolTaskQueueLength(length int) TaskPoolOption {
	return func(o *TaskPoolOptions) {
		o.tQLen = length
	}
}

// @number is the task queue number
func WithTaskPoolTaskQueueNumber(number int) TaskPoolOption {
	return func(o *TaskPoolOptions) {
		o.tQNumber = number
	}
}

/////////////////////////////////////////
// Task Pool
/////////////////////////////////////////

// task t
type task func()

// task pool: manage task ts
type TaskPool struct {
	TaskPoolOptions

	idx    uint32 // round robin index
	qArray []chan task
	wg     sync.WaitGroup

	once sync.Once
	done chan struct{}
}

// build a task pool
func NewTaskPool(opts ...TaskPoolOption) *TaskPool {
	var tOpts TaskPoolOptions
	for _, opt := range opts {
		opt(&tOpts)
	}

	tOpts.validate()

	p := &TaskPool{
		TaskPoolOptions: tOpts,
		qArray:          make([]chan task, tOpts.tQNumber),
		done:            make(chan struct{}),
	}

	for i := 0; i < p.tQNumber; i++ {
		p.qArray[i] = make(chan task, p.tQLen)
	}
	p.start()

	return p
}

// start task pool
func (p *TaskPool) start() {
	for i := 0; i < p.tQPoolSize; i++ {
		p.wg.Add(1)
		workerID := i
		q := p.qArray[workerID%p.tQNumber]
		p.safeRun(workerID, q)
	}
}

func (p *TaskPool) safeRun(workerID int, q chan task) {
	gxruntime.GoSafely(nil, false,
		func() {
			err := p.run(int(workerID), q)
			if err != nil {
				// log error to stderr
				log.Printf("gost/TaskPool.run error: %s", err.Error())
			}
		},
		nil,
	)
}

// worker
func (p *TaskPool) run(id int, q chan task) error {
	defer p.wg.Done()

	var (
		ok bool
		t  task
	)

	for {
		select {
		case <-p.done:
			if 0 < len(q) {
				return fmt.Errorf("task worker %d exit now while its task buffer length %d is greater than 0",
					id, len(q))
			}

			return nil

		case t, ok = <-q:
			if ok {
				t()
			}
		}
	}
}

// AddTask wait idle worker add task
// return false when the pool is stop
func (p *TaskPool) AddTask(t task) (ok bool) {
	idx := atomic.AddUint32(&p.idx, 1)
	id := idx % uint32(p.tQNumber)

	select {
	case <-p.done:
		return false
	default:
		p.qArray[id] <- t
		return true
	}
}

// AddTaskAlways add task to queues or do it immediately
func (p *TaskPool) AddTaskAlways(t task) {
	id := atomic.AddUint32(&p.idx, 1) % uint32(p.tQNumber)

	select {
	case p.qArray[id] <- t:
		return
	default:
		p.goSafely(t)
	}
}

// AddTaskBalance add task to idle queue
// do it immediately when no idle queue
func (p *TaskPool) AddTaskBalance(t task) {
	length := len(p.qArray)

	// try len/2 times to lookup idle queue
	for i := 0; i < length/2; i++ {
		select {
		case p.qArray[rand.Intn(length)] <- t:
			return
		default:
			continue
		}
	}

	p.goSafely(t)
}

func (p *TaskPool) goSafely(fn func()) {
	gxruntime.GoSafely(nil, false, fn, nil)
}

// stop all tasks
func (p *TaskPool) stop() {
	select {
	case <-p.done:
		return
	default:
		p.once.Do(func() {
			close(p.done)
		})
	}
}

// check whether the session has been closed.
func (p *TaskPool) IsClosed() bool {
	select {
	case <-p.done:
		return true

	default:
		return false
	}
}

func (p *TaskPool) Close() {
	p.stop()
	p.wg.Wait()
	for i := range p.qArray {
		close(p.qArray[i])
	}
}
