package shm

import (
	"testing"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"strconv"
)

func TestEntry(t *testing.T) {
	// 4096 * 100
	zone := InitMetricsZone("TestShmZone", 4096*100)
	defer zone.Detach()

	entry1, _ := defaultZone.alloc("entry_test_1")

	expected := 10000
	cpu := runtime.NumCPU()
	wg := sync.WaitGroup{}
	wg.Add(cpu * 2)
	for i := 0; i < cpu; i++ {
		go func() {
			for j := 0; j < expected/cpu; j++ {
				atomic.AddInt64(&entry1.value, 1)
			}
			wg.Done()
		}()
	}

	entry2, _ := defaultZone.alloc("entry_test_2")
	for i := 0; i < cpu; i++ {
		go func() {
			for j := 0; j < expected/cpu; j++ {
				atomic.AddInt64(&entry2.value, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	fmt.Printf("entry1 %+v\n", entry1)
	fmt.Printf("entry2 %+v\n", entry2)

}

func BenchmarkMultiEntry(b *testing.B) {
	zone := InitMetricsZone("TestMultiEntry", 10*1024*1024)
	defer zone.Detach()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i ++ {
		defaultZone.alloc("bench_" + strconv.Itoa(i))
	}
}
