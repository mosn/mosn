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

// WorkerFunc is called by the goroutine of the ShardWorkerPool and assumed never return in normal case.
type WorkerFunc func(shard int, jobCh <-chan interface{})

// ShardJob represents a job with its shard source.
type ShardJob interface {
	// Source get the job identifier for sharding.
	Source() uint32
}

// ShardWorkerPool provides a pool for goroutines, the actual goroutines themselves are assumed permanent running
// and waiting for the incoming jobs from job channel. Its behaviour is specified by the
// :ref:`WorkerFunc<sync.WorkerFunc>`.
//
// In order to prevent excessive lock contention, the ShardWorkerPool also implements sharding of its underlying
// worker jobs. Source value, which can be used to calculate the actual shard(goroutine), is required when jobs are committed
// to the ShardWorkerPool. Jobs in the same shard are serial executed in FIFO order.
//
// shard goroutine respawn is performed while panic during the WorkerFunc execution.
type ShardWorkerPool interface {
	// Init initializes the pool.
	Init()

	// Shard get the real shard of giving source, this may helps in case like global object accessing.
	Shard(source uint32) uint32

	// Offer puts the job into the corresponding shard and execute it.
	Offer(job ShardJob, block bool)
}

// WorkerPool provides a pool for goroutines
type WorkerPool interface {

	// Schedule try to acquire pooled worker goroutine to execute the specified task,
	// this method would block if no worker goroutine is available
	Schedule(task func())

	// Schedule try to acquire pooled worker goroutine to execute the specified task first,
	// but would not block if no worker goroutine is available. A temp goroutine will be created for task execution.
	ScheduleAlways(task func())

	ScheduleAuto(task func())
}
