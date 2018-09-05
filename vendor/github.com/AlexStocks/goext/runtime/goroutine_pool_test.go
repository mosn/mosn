// +build !leak
package gxruntime

import (
	"sync"
	"testing"
	"time"
)

// go test -v -bench ".*"
func TestBasicAPI(t *testing.T) {
	gp := NewGoroutinePool(time.Second)
	// cover alloc()
	gp.Go(func() {})
	// cover put()
	// cover get()
	gp.Go(func() {})
	gp.Close()
}

func TestGC(t *testing.T) {
	gp := NewGoroutinePool(200 * time.Millisecond)
	for i := 0; i < 100; i++ {
		idx := i
		gp.Go(func() {
			time.Sleep(time.Duration(idx+1) * time.Millisecond)
		})
	}
	gp.Close()
	time.Sleep(1e9)
	gp.Lock()
	count := gp.count
	gp.Unlock()
	if count != 0 {
		t.Errorf("all goroutines should be recycled, count:%d\n", count)
	}
}

func TestRace(t *testing.T) {
	gp := NewGoroutinePool(200 * time.Millisecond)
	begin := make(chan struct{})
	for i := 0; i < 50; i++ {
		idxI := i
		go func() {
			<-begin
			for i := 0; i < 10; i++ {
				idxJ := i
				res := gp.Go(func() {})
				if res != nil {
					t.Logf("fail to start work %d-%d", idxI, idxJ)
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}
	close(begin)

	gp.Close()
	time.Sleep(1e9)
	gp.Lock()
	count := gp.count
	gp.Unlock()
	if count != 0 {
		t.Errorf("all goroutines should be recycled, count:%d\n", count)
	}
	t.Logf("work req number:%d, work finish number:%d", gp.workStartNum, gp.workFinNum)
}

// go test -v -bench GoPool -run=^a
func BenchmarkGoPool(b *testing.B) {
	gp := NewGoroutinePool(10 * time.Second)
	for i := 0; i < b.N; i++ {
		gp.Go(func() {})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gp.Go(dummy)
	}
	b.StopTimer()

	gp.Close()
}

func BenchmarkGo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go dummy()
	}
}

func dummy() {
}

func BenchmarkMorestackPool(b *testing.B) {
	gp := NewGoroutinePool(5 * time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gp.Go(func() {
			morestack(false)
		})
	}
	b.StopTimer()

	gp.Close()
}

func BenchmarkMoreStack(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			morestack(false)
			wg.Done()
		}()
		wg.Wait()
	}
}

func morestack(f bool) {
	var stack [8 * 1024]byte
	if f {
		for i := 0; i < len(stack); i++ {
			stack[i] = 'a'
		}
	}
}
