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
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/log"
	"time"
)

type TestJob struct {
	i uint32
}

func (t *TestJob) Source() uint32 {
	return t.i
}

// TestJobOrder test worker pool's event dispatch functionality, which should ensure the FIFO order
func TestJobOrder(t *testing.T) {
	log.InitDefaultLogger("stdout", log.DEBUG)
	shardEvents := 512
	wg := sync.WaitGroup{}

	consumer := func(shard int, jobChan <-chan interface{}) {
		prev := 0
		count := 0

		for job := range jobChan {
			if testJob, ok := job.(*TestJob); ok {
				if int(testJob.i) <= prev {
					t.Errorf("unexpected event order, shard %d, prev %d, curr %d", shard, prev, testJob.i)
					wg.Done()
					return
				}

				prev = int(testJob.i)
				count++

				if count >= shardEvents {
					wg.Done()
					return
				}
			}
		}

	}

	shardsNum := runtime.NumCPU()
	// shard cap is 64
	pool, _ := NewShardWorkerPool(shardsNum*64, shardsNum, consumer)
	pool.Init()

	// multi goroutine offer is not guaranteed FIFO order, because race condition may happen in Offer method
	// so we let the producer and consumer to be one-to-one relation.
	for i := 0; i < shardsNum; i++ {
		wg.Add(1)
		counter := uint32(i)
		go func() {
			for j := 0; j < shardEvents; j++ {
				pool.Offer(&TestJob{i: atomic.AddUint32(&counter, uint32(shardsNum))})
			}
		}()
	}

	wg.Wait()

	// wait flush end for codecov
	for atomic.LoadUint32(&pool.(*shardWorkerPool).schedule) != 0 {
		time.Sleep(time.Millisecond * 10)
	}
}

func eventProcess(b *testing.B) {
	shardEvents := 512
	wg := sync.WaitGroup{}

	consumer := func(shard int, jobChan <-chan interface{}) {
		prev := 0
		count := 0

		for job := range jobChan {
			if testJob, ok := job.(*TestJob); ok {
				if int(testJob.i) <= prev {
					b.Errorf("unexpected event order, shard %d, prev %d, curr %d", shard, prev, testJob.i)
					wg.Done()
					return
				}

				prev = int(testJob.i)
				count++

				if count >= shardEvents {
					wg.Done()
					return
				}
			}
		}

	}

	shardsNum := runtime.NumCPU()
	// shard cap is 64
	pool, _ := NewShardWorkerPool(shardsNum*64, shardsNum, consumer)
	pool.Init()

	// multi goroutine offer is not guaranteed FIFO order, because race condition may happen in Offer method
	// so we let the producer and consumer to be one-to-one relation.
	for i := 0; i < shardsNum; i++ {
		wg.Add(1)
		counter := uint32(i)
		go func() {
			for j := 0; j < shardEvents; j++ {
				pool.Offer(&TestJob{i: atomic.AddUint32(&counter, uint32(shardsNum))})
			}
		}()
	}

	wg.Wait()
}

func BenchmarkShardWorkerPool(b *testing.B) {
	log.InitDefaultLogger("stdout", log.ERROR)

	for i := 0; i < b.N; i++ {
		eventProcess(b)
	}
}

// Implements from https://medium.com/capital-one-developers/building-an-unbounded-channel-in-go-789e175cd2cd
func MakeInfinite() (chan<- interface{}, <-chan interface{}) {
	in := make(chan interface{})
	out := make(chan interface{})

	go func() {
		var inQueue []interface{}
		outCh := func() chan interface{} {
			if len(inQueue) == 0 {
				return nil
			}
			return out
		}

		curVal := func() interface{} {
			if len(inQueue) == 0 {
				return nil
			}
			return inQueue[0]
		}

		for len(inQueue) > 0 || in != nil {
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
				} else {
					inQueue = append(inQueue, v)
				}
			case outCh() <- curVal():
				inQueue = inQueue[1:]
			}
		}
		close(out)
	}()
	return in, out
}

func eventProcessWithUnboundedChannel(b *testing.B) {
	shardEvents := 512
	wg := sync.WaitGroup{}

	consumer := func(shard int, jobChan <-chan interface{}) {
		prev := 0
		count := 0

		for job := range jobChan {
			if testJob, ok := job.(*TestJob); ok {
				if int(testJob.i) <= prev {
					b.Errorf("unexpected event order, shard %d, prev %d, curr %d", shard, prev, testJob.i)
					wg.Done()
					return
				}

				prev = int(testJob.i)
				count++

				if count >= shardEvents {
					wg.Done()
					return
				}
			}
		}

	}

	shardsNum := runtime.NumCPU()

	// multi goroutine offer is not guaranteed FIFO order, because race condition may happen in Offer method
	// so we let the producer and consumer to be one-to-one relation.
	for i := 0; i < shardsNum; i++ {
		wg.Add(1)
		counter := uint32(i)
		in, out := MakeInfinite()

		go func() {
			for j := 0; j < shardEvents; j++ {
				in <- &TestJob{i: atomic.AddUint32(&counter, uint32(shardsNum))}
			}
		}()
		go func(shard int) {
			consumer(shard, out)
		}(i)
	}

	wg.Wait()
}

func BenchmarkUnboundChannel(b *testing.B) {
	log.InitDefaultLogger("stdout", log.ERROR)

	for i := 0; i < b.N; i++ {
		eventProcessWithUnboundedChannel(b)
	}
}
