package gxtime

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	"github.com/stretchr/testify/assert"
)

func TestNewTimerWheel(t *testing.T) {
	var (
		index int
		wheel *TimerWheel
		cw    CountWatch
	)

	wheel = NewTimerWheel()
	defer func() {
		fmt.Println("timer costs:", cw.Count()/1e6, "ms")
		wheel.Stop()
	}()

	cw.Start()
	for {
		select {
		case <-wheel.After(TimeMillisecondDuration(100)):
			index++
			if index >= 10 {
				return
			}
		}
	}
}

func TestAfter(t *testing.T) {
	var (
		wheel *TimerWheel
		wg    sync.WaitGroup
	)
	wheel = NewTimerWheel()

	defer wheel.Stop()

	f := func(d time.Duration, num int) {
		var (
			cw    CountWatch
			index int
		)

		defer func() {
			gxlog.CInfo("duration %d loop %d, timer costs:%dms", d, num, cw.Count()/1e6)
			gxlog.CInfo("in timer func, timer number:%d", wheel.TimerNumber())
			wg.Done()
		}()

		cw.Start()
		for {
			select {
			case <-wheel.After(d):
				index++
				if index >= num {
					return
				}
			}
		}
	}

	wg.Add(6)
	go f(TimeSecondDuration(1.5), 15)
	go f(TimeSecondDuration(2.510), 10)
	go f(TimeSecondDuration(1.5), 40)
	go f(TimeSecondDuration(0.15), 200)
	go f(TimeSecondDuration(3), 20)
	go f(TimeSecondDuration(63), 1)

	time.Sleep(TimeSecondDuration(0.01))
	assert.Equalf(t, 6, defaultTimerWheel.TimerNumber(), "")
	wg.Wait()
}

func TestAfterFunc(t *testing.T) {
	var (
		wg sync.WaitGroup
		cw CountWatch
	)

	Init()

	f := func() {
		defer wg.Done()
		gxlog.CInfo("timer costs:%dms", cw.Count()/1e6)
		gxlog.CInfo("in timer func, timer number:%d", defaultTimerWheel.TimerNumber())
	}

	wg.Add(3)
	cw.Start()
	AfterFunc(TimeSecondDuration(0.5), f)
	AfterFunc(TimeSecondDuration(1.5), f)
	AfterFunc(TimeSecondDuration(61.5), f)

	time.Sleep(TimeSecondDuration(0.01))
	assert.Equalf(t, 3, defaultTimerWheel.TimerNumber(), "")
	wg.Wait()
}

func TestTimer_Reset(t *testing.T) {
	var (
		timer *Timer
		wg    sync.WaitGroup
		cw    CountWatch
	)

	Init()

	f := func() {
		defer wg.Done()
		gxlog.CInfo("timer costs:%dms", cw.Count()/1e6)
		gxlog.CInfo("in timer func, timer number:%d", defaultTimerWheel.TimerNumber())
	}

	wg.Add(1)
	cw.Start()
	timer = AfterFunc(TimeSecondDuration(1.5), f)
	timer.Reset(TimeSecondDuration(3.5))

	time.Sleep(TimeSecondDuration(0.01))
	assert.Equalf(t, 1, defaultTimerWheel.TimerNumber(), "")
	wg.Wait()
}

func TestTimer_Stop(t *testing.T) {
	var (
		timer *Timer
		cw    CountWatch
	)

	Init()

	f := func() {
		gxlog.CInfo("timer costs:%dms", cw.Count()/1e6)
	}

	timer = AfterFunc(TimeSecondDuration(4.5), f)
	// 添加是异步进行的，所以sleep一段时间再去检测timer number
	time.Sleep(1e9)
	assert.Equalf(t, 1, defaultTimerWheel.TimerNumber(), "before stop")
	timer.Stop()
	// 删除是异步进行的，所以sleep一段时间再去检测timer number
	time.Sleep(1e9)

	time.Sleep(TimeSecondDuration(0.01))
	assert.Equalf(t, 0, defaultTimerWheel.TimerNumber(), "after stop")
	time.Sleep(3e9)
}
