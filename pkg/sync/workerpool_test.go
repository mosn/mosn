package sync

import (
	"testing"
	"runtime"
	"github.com/alipay/sofa-mosn/pkg/log"
	"sync/atomic"
	"sync"
)

type TestJob struct {
	i uint32
}

func (t *TestJob) Source() int {
	return int(t.i)
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
	for i := 0; i < shardsNum; i ++ {
		wg.Add(1)
		counter := uint32(i )
		go func() {
			for j := 0; j < shardEvents; j++ {
				pool.Offer(&TestJob{i: atomic.AddUint32(&counter, uint32(shardsNum))})
			}
		}()
	}

	wg.Wait()
}
