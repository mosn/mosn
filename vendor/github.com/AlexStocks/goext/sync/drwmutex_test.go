// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// refers to github.com/jonhoo/drwmutex
package gxsync

import (
	//"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"testing"
	"time"
)

const (
	SYNC int = iota
	DRWM     = iota
	END      = iota
)

type L interface {
	Lock()
	Unlock()
	RLocker() sync.Locker
}

func TestNewDRWMutex(t *testing.T) {
	//cpuprofile := flag.Bool("cpuprofile", false, "enable CPU profiling")
	//locks := flag.Uint64("i", 10000, "Number of iterations to perform")
	//write := flag.Float64("p", 0.0001, "Probability of write locks")
	//wwork := flag.Int("w", 1, "Amount of work for each writer")
	//rwork := flag.Int("r", 100, "Amount of work for each reader")
	//readers := flag.Int("n", runtime.GOMAXPROCS(0), "Total number of readers")
	//checkcpu := flag.Uint64("c", 100, "Update CPU estimate every n iterations")
	//flag.Parse()
	cpuprofile := true
	locks := 10000
	write := 0.001
	wwork := 2
	rwork := 100
	readers := 4
	checkcpu := 2

	readers_per_core := readers / runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	var mx L

	for l := 0; l < END; l++ {
		var o *os.File
		if cpuprofile {
			if o != nil {
				pprof.StopCPUProfile()
				o.Close()
			}

			o, _ := os.Create(fmt.Sprintf("rw%d.out", l))
			pprof.StartCPUProfile(o)
		}

		switch l {
		case SYNC:
			mx = new(sync.RWMutex)
		case DRWM:
			mx = NewDRWMutex()
		}

		start := time.Now()
		for n := 0; n < runtime.GOMAXPROCS(0); n++ {
			for r := 0; r < readers_per_core; r++ {
				wg.Add(1)
				// go func() {
				func() {
					defer wg.Done()
					rmx := mx.RLocker()
					r := rand.New(rand.NewSource(rand.Int63()))
					for n := uint64(0); n < uint64(locks); n++ {
						if l != SYNC && checkcpu != 0 && n%uint64(checkcpu) == 0 {
							rmx = mx.RLocker()
						}
						if r.Float64() < write {
							mx.Lock()
							x := 0
							for i := 0; i < wwork; i++ {
								x++
							}
							_ = x
							mx.Unlock()
						} else {
							rmx.Lock()
							x := 0
							for i := 0; i < rwork; i++ {
								x++
							}
							_ = x
							rmx.Unlock()
						}
					}
				}()
			}
		}
		wg.Wait()
		end := time.Now()

		diff := end.Sub(start)
		t.Logf(fmt.Sprintf("mx%d", l+1), runtime.GOMAXPROCS(0), readers,
			locks, write, wwork, rwork, checkcpu, diff.Seconds(), diff)
	}
}
