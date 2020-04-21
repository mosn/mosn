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
	"fmt"
	"runtime/debug"

	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/utils"
)

const (
	maxRespwanTimes = 1 << 6
)

type shard struct {
	index        int
	respawnTimes uint32
	jobChan      chan interface{}
}

type shardWorkerPool struct {
	// workerFunc should never exit, always try to acquire jobs from jobs channel
	workerFunc WorkerFunc
	shards     []*shard
	numShards  int
}

// NewShardWorkerPool creates a new shard worker pool.
func NewShardWorkerPool(size int, numShards int, workerFunc WorkerFunc) (ShardWorkerPool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("worker pool size too small: %d", size)
	}
	if size < numShards {
		numShards = size
	}
	shardCap := size / numShards
	shards := make([]*shard, numShards)
	for i := range shards {
		shards[i] = &shard{
			index:   i,
			jobChan: make(chan interface{}, shardCap),
		}
	}
	return &shardWorkerPool{
		workerFunc: workerFunc,
		shards:     shards,
		numShards:  numShards,
	}, nil
}

func (pool *shardWorkerPool) Init() {
	for i := range pool.shards {
		pool.spawnWorker(pool.shards[i])
	}
}

func (pool *shardWorkerPool) Shard(source uint32) uint32 {
	return source % uint32(pool.numShards)
}

func (pool *shardWorkerPool) Offer(job ShardJob, block bool) {
	// use shard to avoid excessive synchronization
	i := pool.Shard(job.Source())
	if block {
		pool.shards[i].jobChan <- job
	} else {
		select {
		case pool.shards[i].jobChan <- job:
		default:
			log.DefaultLogger.Errorf("[syncpool] jobChan over full")
		}
	}
}

func (pool *shardWorkerPool) spawnWorker(shard *shard) {
	utils.GoWithRecover(func() {
		pool.workerFunc(shard.index, shard.jobChan)
	}, func(r interface{}) {
		if shard.respawnTimes < maxRespwanTimes {
			shard.respawnTimes++
			pool.spawnWorker(shard)
		}
	})
}

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
			log.DefaultLogger.Alertf("syncpool", "[syncpool] panic %v\n%s", p, string(debug.Stack()))
		}
		<-p.sem
	}()
	for {
		task()
		task = <-p.work
	}
}
