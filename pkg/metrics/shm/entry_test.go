package shm

import (
	"testing"
	"github.com/alipay/sofa-mosn/pkg/shm"
	"unsafe"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"strconv"
)

func TestEntry(t *testing.T) {
	// 4096 * 100
	span, _ := shm.Alloc("TestShmZone", 4096*100)
	defer shm.DeAlloc(span)

	size := unsafe.Sizeof(metricsEntry{})

	entryPtr1, _ := span.Alloc(int(size))
	entry1 := (*metricsEntry)(unsafe.Pointer(entryPtr1))
	entry1.assignName([]byte("entry_test_1"))

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

	entryPtr2, _ := span.Alloc(int(size))
	entry2 := (*metricsEntry)(unsafe.Pointer(entryPtr2))
	entry2.assignName([]byte("entry_test_2"))
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
	//f, _ := os.OpenFile("mem.prof", os.O_RDWR|os.O_CREATE, 0644)
	//defer f.Close()

	zone, err := NewSharedMetrics("TestMultiEntry", 100*1024*1024)
	if err != nil {
		b.Fatal("open shared memory for metrics failed:", err)
	}
	defer zone.Free()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i ++ {
		zone.AllocEntry("bench_" + strconv.Itoa(i))
	}

}
