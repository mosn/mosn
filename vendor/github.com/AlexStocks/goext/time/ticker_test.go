package gxtime

import (
	//"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	"github.com/stretchr/testify/assert"
)

// 每个函数单独进行测试，否则timer number会不准确，因为ticker相关的timer会用于运行下去
func TestNewTicker(t *testing.T) {
	var (
		num     int
		wg      sync.WaitGroup
		xassert *assert.Assertions
	)

	Init()

	f := func(d time.Duration, num int) {
		var (
			cw    CountWatch
			index int
		)
		defer func() {
			gxlog.CInfo("duration %d loop %d, timer costs:%dms", d, num, cw.Count()/1e6)
			wg.Done()
		}()

		cw.Start()

		for range NewTicker(d).C {
			index++
			//gxlog.CInfo("idx:%d, tick:%s", index, t)
			if index >= num {
				return
			}
		}
	}

	num = 6
	xassert = assert.New(t)
	wg.Add(num)
	go f(TimeSecondDuration(1.5), 10)
	go f(TimeSecondDuration(2.51), 10)
	go f(TimeSecondDuration(1.5), 40)
	go f(TimeSecondDuration(0.15), 200)
	go f(TimeSecondDuration(3), 20)
	go f(TimeSecondDuration(63), 1)
	time.Sleep(TimeSecondDuration(0.001))
	xassert.Equal(defaultTimerWheel.TimerNumber(), num, "")
	wg.Wait()
}

func TestTick(t *testing.T) {
	var (
		num     int
		wg      sync.WaitGroup
		xassert *assert.Assertions
	)

	Init()

	f := func(d time.Duration, num int) {
		var (
			cw    CountWatch
			index int
		)
		defer func() {
			gxlog.CInfo("duration %d loop %d, timer costs:%dms", d, num, cw.Count()/1e6)
			wg.Done()
		}()

		cw.Start()

		// for t := range Tick(d)
		for range Tick(d) {
			index++
			//gxlog.CInfo("idx:%d, tick:%s", index, t)
			if index >= num {
				return
			}
		}
	}

	num = 6
	xassert = assert.New(t)
	wg.Add(num)
	go f(TimeSecondDuration(1.5), 10)
	go f(TimeSecondDuration(2.51), 10)
	go f(TimeSecondDuration(1.5), 40)
	go f(TimeSecondDuration(0.15), 200)
	go f(TimeSecondDuration(3), 20)
	go f(TimeSecondDuration(63), 1)
	time.Sleep(0.001e9)
	xassert.Equal(defaultTimerWheel.TimerNumber(), num, "")
	wg.Wait()
}

func TestTickFunc(t *testing.T) {
	var (
		num     int
		cw      CountWatch
		xassert *assert.Assertions
	)

	Init()

	f := func() {
		gxlog.CInfo("timer costs:%dms", cw.Count()/1e6)
	}

	num = 3
	xassert = assert.New(t)
	cw.Start()
	TickFunc(TimeSecondDuration(0.5), f)
	TickFunc(TimeSecondDuration(1.3), f)
	TickFunc(TimeSecondDuration(61.5), f)
	time.Sleep(62e9)
	xassert.Equal(defaultTimerWheel.TimerNumber(), num, "")
}

func TestTicker_Reset(t *testing.T) {
	var (
		ticker  *Ticker
		wg      sync.WaitGroup
		cw      CountWatch
		xassert *assert.Assertions
	)

	Init()

	f := func() {
		defer wg.Done()
		gxlog.CInfo("timer costs:%dms", cw.Count()/1e6)
		gxlog.CInfo("in timer func, timer number:%d", defaultTimerWheel.TimerNumber())
	}

	xassert = assert.New(t)
	wg.Add(1)
	cw.Start()
	ticker = TickFunc(TimeSecondDuration(1.5), f)
	ticker.Reset(TimeSecondDuration(3.5))
	time.Sleep(TimeSecondDuration(0.001))
	xassert.Equal(defaultTimerWheel.TimerNumber(), 1, "")
	wg.Wait()
}

func TestTicker_Stop(t *testing.T) {
	var (
		ticker  *Ticker
		cw      CountWatch
		xassert assert.Assertions
	)

	Init()

	f := func() {
		gxlog.CInfo("timer costs:%dms", cw.Count()/1e6)
	}

	cw.Start()
	ticker = TickFunc(TimeSecondDuration(4.5), f)
	// 添加是异步进行的，所以sleep一段时间再去检测timer number
	time.Sleep(TimeSecondDuration(0.001))
	xassert.Equal(defaultTimerWheel.TimerNumber(), 1, "")
	time.Sleep(TimeSecondDuration(5))
	ticker.Stop()
	// 删除是异步进行的，所以sleep一段时间再去检测timer number
	time.Sleep(TimeSecondDuration(0.001))
	xassert.Equal(defaultTimerWheel.TimerNumber(), 0, "")
}
