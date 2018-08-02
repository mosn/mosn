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
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/alipay/sofa-mosn/pkg/log"
)

const (
	maxRespwanTimes = 1 << 6
)

type shard struct {
	sync.Mutex

	index        int
	respawnTimes uint32

	// job
	jobChan  chan interface{}
	jobQueue []interface{}

	// control
	ctrlChan  chan interface{}
	ctrlQueue []interface{}
}

type shardWorkerPool struct {
	sync.Mutex

	// workerFunc should never exit, always try to acquire jobs from jobs channel
	workerFunc WorkerFunc
	shards     []*shard

	numShards int
	// represents whether job scheduler for queued jobs is started or not
	schedule uint32
}

// NewPooledWorkerPool creates a new shard worker pool.
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
			index:    i,
			jobChan:  make(chan interface{}, shardCap),
			ctrlChan: make(chan interface{}, shardCap),
		}
	}

	return &shardWorkerPool{
		workerFunc: workerFunc,
		shards:     shards,
		numShards:  numShards,
	}, nil
}

func (p *shardWorkerPool) Init() {
	for i := range p.shards {
		p.spawnWorker(p.shards[i])
	}
}

func (p *shardWorkerPool) Shard(source int) int {
	return source % p.numShards
}

func (p *shardWorkerPool) Offer(job ShardJob) {
	// use shard to avoid excessive synchronization
	i := p.Shard(job.Source())
	shard := p.shards[i]

	// put jobs to the jobChan or jobQueue, which determined by the shard workload
	shard.Lock()
	switch job.Type() {
	case NORMAL:
		select {
		case shard.jobChan <- job:
		default:
			shard.jobQueue = append(shard.jobQueue, job)
			// schedule flush if
			if atomic.CompareAndSwapUint32(&p.schedule, 0, 1) {
				p.flush()
			}
		}
	case CONTROL:
		select {
		case shard.ctrlChan <- job:
		default:
			shard.ctrlQueue = append(shard.ctrlQueue, job)
			// schedule flush if
			if atomic.CompareAndSwapUint32(&p.schedule, 0, 1) {
				p.flush()
			}
		}
	default:
		log.DefaultLogger.Errorf("Unknown job type %d", job.Type())
	}

	shard.Unlock()
}

func (pool *shardWorkerPool) spawnWorker(shard *shard) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.DefaultLogger.Errorf("worker panic %v", p)
				debug.PrintStack()

				//try respawn worker
				if shard.respawnTimes < maxRespwanTimes {
					shard.respawnTimes++
					pool.spawnWorker(shard)
				}
			}
		}()

		pool.workerFunc(shard.index, shard.jobChan, shard.ctrlChan)
	}()
}

func (p *shardWorkerPool) flush() {
	go func() {

		for {
			clear := true

			for i := range p.shards {
				shard := p.shards[i]

				shard.Lock()

				// flush controls
				pendingCtrl, writeCtrl := flush(&shard.ctrlQueue, shard.ctrlChan)
				if writeCtrl > 0 {
					log.DefaultLogger.Debugf("flush %d control to shard %d", writeCtrl, i)
				}

				// flush jobs
				pendingJob, writeJob := flush(&shard.jobQueue, shard.jobChan)
				if writeJob > 0 {
					log.DefaultLogger.Debugf("flush %d job to shard %d", writeJob, i)
				}

				if clear && (pendingCtrl > 0 || pendingJob > 0) {
					clear = false
				}

				shard.Unlock()
			}

			// all shards' job queue are clear, stop flush goroutine
			if clear {
				break
			}

			// wait for next schedule
			runtime.Gosched()
		}

		// end flush schedule
		atomic.CompareAndSwapUint32(&p.schedule, 1, 0)
	}()
}

func flush(queue *[]interface{}, ch chan interface{}) (pending, write int) {
	pending = len(*queue)
	slots := cap(ch) - len(ch)

	// the number we can write is determined by the minimal of pending jobs and available chan data slots
	write = min(pending, slots)

	if write > 0 {
		for j := 0; j < write; j++ {
			ch <- (*queue)[j]
		}
		*queue = (*queue)[write:]
	}
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
