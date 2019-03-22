package rtimer

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

//TestTimeWheel test timewheel.
func TestTimeWheel(t *testing.T) {
	var deviation int64
	var num int64 = 100
	sleepTime := time.Millisecond * 20000

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			start := time.Now().UnixNano()
			<-After(sleepTime)
			end := time.Now().UnixNano()
			d := (end - start) - int64(sleepTime)
			if d >= 0 {
				deviation += d
			} else {
				deviation -= d
			}
			wg.Done()
		}(i)
		time.Sleep(time.Millisecond * 100)
	}
	wg.Wait()
	fmt.Println(float64(deviation) / float64(num*int64(sleepTime)))
}

//BenchmarkTimeWheel benchmarks timewheel.
func BenchmarkTimeWheel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		After(time.Millisecond * 100)
	}
}

//BenchmarkTimeBase benchmark origin timer.
func BenchmarkTimeBase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		time.After(time.Millisecond * 100)
	}
}
