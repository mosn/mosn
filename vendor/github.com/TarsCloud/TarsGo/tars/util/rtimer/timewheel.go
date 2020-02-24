package rtimer

import (
	"sync"
	"time"
)

var (
	timerMap map[time.Duration]*TimeWheel
	mapLock  = &sync.Mutex{}
	accuracy = 20 // means max 1/20 deviation
)

func init() {
	timerMap = make(map[time.Duration]*TimeWheel)
}

func After(t time.Duration) <-chan struct{} {
	mapLock.Lock()
	defer mapLock.Unlock()
	if v, ok := timerMap[t]; ok {
		return v.After(t)
	}
	v := NewTimeWheel(t/time.Duration(accuracy), accuracy+1)
	timerMap[t] = v
	return v.After(t)
}

func SetAccuracy(a int) {
	accuracy = a
}

type TimeWheel struct {
	lock sync.Mutex
	t    time.Duration
	maxT time.Duration

	ticker *time.Ticker

	timeWheel []chan struct{}
	currPos   int
}

func NewTimeWheel(t time.Duration, size int) *TimeWheel {
	tw := &TimeWheel{t: t, maxT: t * time.Duration(size)}

	tw.timeWheel = make([]chan struct{}, size)
	for i := range tw.timeWheel {
		tw.timeWheel[i] = make(chan struct{})
	}
	tw.ticker = time.NewTicker(t)
	go tw.run()
	return tw
}

func (tw *TimeWheel) Stop() {
	tw.ticker.Stop()
}

func (tw *TimeWheel) After(timeout time.Duration) <-chan struct{} {
	if timeout >= tw.maxT {
		panic("timeout is bigger than maxT")
	}

	pos := int(timeout / tw.t)
	if 0 < pos {
		pos--
	}
	tw.lock.Lock()
	pos = (tw.currPos + pos) % len(tw.timeWheel)
	c := tw.timeWheel[pos]
	tw.lock.Unlock()
	return c
}

func (tw *TimeWheel) run() {
	for range tw.ticker.C {
		tw.lock.Lock()
		oldestC := tw.timeWheel[tw.currPos]
		tw.timeWheel[tw.currPos] = make(chan struct{})
		tw.currPos = (tw.currPos + 1) % len(tw.timeWheel)
		tw.lock.Unlock()
		close(oldestC)
	}
}
