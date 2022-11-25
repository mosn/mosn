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

package sync

import (
	"runtime/debug"

	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/utils"
)

type workerPool struct {
	work chan func()
	sem  chan struct{}
}

// NewWorkerPool create a worker pool
func NewWorkerPool(size int) WorkerPool {
	return &workerPool{
		work: make(chan func()),
		sem:  make(chan struct{}, size),
	}
}

func (p *workerPool) Schedule(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	}
}

func (p *workerPool) ScheduleAlways(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	default:
		// new temp goroutine for task execution
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[syncpool] workerpool new goroutine")
		}
		utils.GoWithRecover(func() {
			task()
		}, nil)
	}
}

func (p *workerPool) ScheduleAuto(task func()) {
	select {
	case p.work <- task:
		return
	default:
	}
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.spawnWorker(task)
	default:
		// new temp goroutine for task execution
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[syncpool] workerpool new goroutine")
		}
		utils.GoWithRecover(func() {
			task()
		}, nil)
	}
}

func (p *workerPool) spawnWorker(task func()) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Alertf("syncpool", "[syncpool] panic %v\n%s", r, string(debug.Stack()))
		}
		<-p.sem
	}()
	for {
		task()
		task = <-p.work
	}
}
