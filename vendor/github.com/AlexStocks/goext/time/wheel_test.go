package gxtime

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// output:
// timer costs: 100002 ms
// --- PASS: TestNewWheel (100.00s)
func TestWheel(t *testing.T) {
	var (
		index int
		wheel *Wheel
		cw    CountWatch
	)
	wheel = NewWheel(TimeMillisecondDuration(100), 20)
	defer func() {
		fmt.Println("timer costs:", cw.Count()/1e6, "ms")
		wheel.Stop()
	}()

	cw.Start()
	for {
		select {
		case <-wheel.After(TimeMillisecondDuration(1000)):
			fmt.Println("loop:", index)
			index++
			if index >= 150 {
				return
			}
		}
	}
}

// output:
// timer costs: 150001 ms
// --- PASS: TestNewWheel2 (150.00s)
func TestWheels(t *testing.T) {
	var (
		wheel *Wheel
		cw    CountWatch
		wg    sync.WaitGroup
	)
	wheel = NewWheel(TimeMillisecondDuration(100), 20)
	defer func() {
		fmt.Println("timer costs:", cw.Count()/1e6, "ms") //
		wheel.Stop()
	}()

	f := func(d time.Duration) {
		defer wg.Done()
		var index int
		for {
			select {
			case <-wheel.After(d):
				fmt.Println("loop:", index, ", interval:", d)
				index++
				if index >= 100 {
					return
				}
			}
		}
	}

	wg.Add(2)
	cw.Start()
	go f(1e9)
	go f(1510e6)
	wg.Wait()
}
