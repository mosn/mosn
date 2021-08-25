/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package gxtime encapsulates some golang.time functions
package gxtime

import (
	"container/list"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

import (
	uatomic "go.uber.org/atomic"
)

var (
	// nolint
	ErrTimeChannelFull = errors.New("timer channel full")
	// nolint
	ErrTimeChannelClosed = errors.New("timer channel closed")
)

// InitDefaultTimerWheel initializes a default timer wheel
func InitDefaultTimerWheel() {
	defaultTimerWheelOnce.Do(func() {
		defaultTimerWheel = NewTimerWheel()
	})
}

func GetDefaultTimerWheel() *TimerWheel {
	return defaultTimerWheel
}

// Now returns the current time.
func Now() time.Time {
	return defaultTimerWheel.Now()
}

////////////////////////////////////////////////
// timer node
////////////////////////////////////////////////

var (
	defaultTimerWheelOnce sync.Once
	defaultTimerWheel     *TimerWheel
	nextID                TimerID
	curGxTime             = time.Now().UnixNano() // current goext time in nanoseconds
)

const (
	maxMS     = 1000
	maxSecond = 60
	maxMinute = 60
	maxHour   = 24
	maxDay    = 31
	// the time accuracy is millisecond.
	minTickerInterval = 10e6
	maxTimerLevel     = 5
)

func msNum(expire int64) int64     { return expire / int64(time.Millisecond) }
func secondNum(expire int64) int64 { return expire / int64(time.Minute) }
func minuteNum(expire int64) int64 { return expire / int64(time.Minute) }
func hourNum(expire int64) int64   { return expire / int64(time.Hour) }
func dayNum(expire int64) int64    { return expire / (maxHour * int64(time.Hour)) }

// TimerFunc defines the time func.
// if the return error is not nil, the related timer will be closed.
type TimerFunc func(ID TimerID, expire time.Time, arg interface{}) error

// TimerID is the id of a timer node
type TimerID = uint64

type timerNode struct {
	ID       TimerID     // node id
	trig     int64       // trigger time
	typ      TimerType   // once or loop
	period   int64       // loop period
	timerRun TimerFunc   // timer func
	arg      interface{} // func arg
}

func newTimerNode(f TimerFunc, typ TimerType, period int64, arg interface{}) *timerNode {
	return &timerNode{
		ID:       atomic.AddUint64(&nextID, 1),
		trig:     atomic.LoadInt64(&curGxTime) + period,
		typ:      typ,
		period:   period,
		timerRun: f,
		arg:      arg,
	}
}

func compareTimerNode(first, second *timerNode) int {
	var ret int

	if first.trig < second.trig {
		ret = -1
	} else if first.trig > second.trig {
		ret = 1
	} else {
		ret = 0
	}

	return ret
}

type timerAction = int64

const (
	TimerActionAdd   timerAction = 1
	TimerActionDel   timerAction = 2
	TimerActionReset timerAction = 3
)

type timerNodeAction struct {
	node   *timerNode
	action timerAction
}

////////////////////////////////////////////////
// timer wheel
////////////////////////////////////////////////

const (
	timerNodeQueueSize = 128
)

var (
	limit   = [maxTimerLevel + 1]int64{maxMS, maxSecond, maxMinute, maxHour, maxDay}
	msLimit = [maxTimerLevel + 1]int64{
		int64(time.Millisecond),
		int64(time.Second),
		int64(time.Minute),
		int64(time.Hour),
		int64(maxHour * time.Hour),
	}
)

// TimerWheel is a timer based on multiple wheels
type TimerWheel struct {
	start  int64                     // start clock
	clock  int64                     // current time in nanosecond
	number uatomic.Int64             // timer node number
	hand   [maxTimerLevel]int64      // clock
	slot   [maxTimerLevel]*list.List // timer list

	enable uatomic.Bool          // timer ready or closed
	timerQ chan *timerNodeAction // timer event notify channel

	once   sync.Once      // for close ticker
	ticker *time.Ticker   // virtual atomic clock
	wg     sync.WaitGroup // gr sync
}

// NewTimerWheel returns a @TimerWheel object.
func NewTimerWheel() *TimerWheel {
	w := &TimerWheel{
		clock: atomic.LoadInt64(&curGxTime),
		// in fact, the minimum time accuracy is 10ms.
		ticker: time.NewTicker(time.Duration(minTickerInterval)),
		timerQ: make(chan *timerNodeAction, timerNodeQueueSize),
	}

	w.enable.Store(true)
	w.start = w.clock

	for i := 0; i < maxTimerLevel; i++ {
		w.slot[i] = list.New()
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		var (
			t          time.Time
			cFlag      bool
			nodeAction *timerNodeAction
			qFlag      bool
		)

	LOOP:
		for {
			if !w.enable.Load() {
				break LOOP
			}

			select {
			case t, cFlag = <-w.ticker.C:
				if !cFlag {
					break LOOP
				}

				atomic.StoreInt64(&curGxTime, t.UnixNano())
				ret := w.timerUpdate(t)
				if ret == 0 {
					w.run()
				}
			case nodeAction, qFlag = <-w.timerQ:
				if !qFlag {
					break LOOP
				}

				// just one w.timerQ channel to ensure the exec sequence of timer event.
				switch {
				case nodeAction.action == TimerActionAdd:
					w.number.Add(1)
					w.insertTimerNode(nodeAction.node)
				case nodeAction.action == TimerActionDel:
					w.number.Add(-1)
					w.deleteTimerNode(nodeAction.node)
				case nodeAction.action == TimerActionReset:
					// log.CInfo("node action:%#v", nodeAction)
					w.resetTimerNode(nodeAction.node)
				default:
					w.number.Add(1)
					w.insertTimerNode(nodeAction.node)
				}
			}
		}
		log.Printf("the timeWheel runner exit, current timer node num:%d", w.number.Load())
	}()
	return w
}

func (w *TimerWheel) output() {
	for idx := range w.slot {
		log.Printf("print slot %d\n", idx)
		// w.slot[idx].Output()
	}
}

// TimerNumber returns the timer obj number in wheel
func (w *TimerWheel) TimerNumber() int {
	return int(w.number.Load())
}

// Now returns the current time
func (w *TimerWheel) Now() time.Time {
	return UnixNano2Time(atomic.LoadInt64(&curGxTime))
}

func (w *TimerWheel) run() {
	var (
		clock         int64
		err           error
		node          *timerNode
		slot          *list.List
		reinsertNodes []*timerNode
	)

	slot = w.slot[0]
	clock = atomic.LoadInt64(&w.clock)
	var next *list.Element
	for e := slot.Front(); e != nil; e = next {
		node = e.Value.(*timerNode)
		if clock < node.trig {
			break
		}

		err = node.timerRun(node.ID, UnixNano2Time(clock), node.arg)
		if err == nil && node.typ == TimerLoop {
			reinsertNodes = append(reinsertNodes, node)
			// w.insertTimerNode(node)
		} else {
			w.number.Add(-1)
		}

		next = e.Next()
		slot.Remove(e)
	}

	for _, reinsertNode := range reinsertNodes {
		reinsertNode.trig += reinsertNode.period
		w.insertTimerNode(reinsertNode)
	}
}

func (w *TimerWheel) insertSlot(idx int, node *timerNode) {
	var (
		pos  *list.Element
		slot *list.List
	)

	slot = w.slot[idx]
	for e := slot.Front(); e != nil; e = e.Next() {
		if compareTimerNode(node, e.Value.(*timerNode)) < 0 {
			pos = e
			break
		}
	}

	if pos != nil {
		slot.InsertBefore(node, pos)
	} else {
		// if slot is empty or @node_ptr is the maximum node
		// in slot, insert it at the last of slot
		slot.PushBack(node)
	}
}

func (w *TimerWheel) deleteTimerNode(node *timerNode) {
	var level int

LOOP:
	for level = range w.slot[:] {
		for e := w.slot[level].Front(); e != nil; e = e.Next() {
			if e.Value.(*timerNode).ID == node.ID {
				w.slot[level].Remove(e)
				// atomic.AddInt64(&w.number, -1)
				break LOOP
			}
		}
	}
}

func (w *TimerWheel) resetTimerNode(node *timerNode) {
	var level int

LOOP:
	for level = range w.slot[:] {
		for e := w.slot[level].Front(); e != nil; e = e.Next() {
			if e.Value.(*timerNode).ID == node.ID {
				n := e.Value.(*timerNode)
				n.trig -= n.period
				n.period = node.period
				n.trig += n.period
				w.slot[level].Remove(e)
				w.insertTimerNode(n)
				break LOOP
			}
		}
	}
}

func (w *TimerWheel) deltaDiff(clock int64) int64 {
	var handTime int64

	for idx, hand := range w.hand[:] {
		handTime += hand * msLimit[idx]
	}

	return clock - w.start - handTime
}

func (w *TimerWheel) insertTimerNode(node *timerNode) {
	var (
		idx  int
		diff int64
	)

	diff = node.trig - atomic.LoadInt64(&w.clock)
	switch {
	case diff <= 0:
		idx = 0
	case dayNum(diff) != 0:
		idx = 4
	case hourNum(diff) != 0:
		idx = 3
	case minuteNum(diff) != 0:
		idx = 2
	case secondNum(diff) != 0:
		idx = 1
	default:
		idx = 0
	}

	w.insertSlot(idx, node)
}

func (w *TimerWheel) timerCascade(level int) {
	var (
		guard bool
		clock int64
		diff  int64
		cur   *timerNode
	)

	clock = atomic.LoadInt64(&w.clock)
	var next *list.Element
	for e := w.slot[level].Front(); e != nil; e = next {
		cur = e.Value.(*timerNode)
		diff = cur.trig - clock
		switch {
		case cur.trig <= clock:
			guard = false
		case level == 1:
			guard = secondNum(diff) > 0
		case level == 2:
			guard = minuteNum(diff) > 0
		case level == 3:
			guard = hourNum(diff) > 0
		case level == 4:
			guard = dayNum(diff) > 0
		}

		if guard {
			break
		}

		next = e.Next()
		w.slot[level].Remove(e)

		w.insertTimerNode(cur)
	}
}

func (w *TimerWheel) timerUpdate(curTime time.Time) int {
	var (
		clock  int64
		now    int64
		idx    int32
		diff   int64
		maxIdx int32
		inc    [maxTimerLevel + 1]int64
	)

	now = curTime.UnixNano()
	clock = atomic.LoadInt64(&w.clock)
	diff = now - clock
	diff += w.deltaDiff(clock)
	if diff < minTickerInterval*0.7 {
		return -1
	}
	atomic.StoreInt64(&w.clock, now)

	for idx = maxTimerLevel - 1; 0 <= idx; idx-- {
		inc[idx] = diff / msLimit[idx]
		diff %= msLimit[idx]
	}

	maxIdx = 0
	for idx = 0; idx < maxTimerLevel; idx++ {
		if 0 != inc[idx] {
			w.hand[idx] += inc[idx]
			inc[idx+1] += w.hand[idx] / limit[idx]
			w.hand[idx] %= limit[idx]
			maxIdx = idx + 1
		}
	}

	for idx = 1; idx < maxIdx; idx++ {
		w.timerCascade(int(idx))
	}

	return 0
}

// Stop stops the ticker
func (w *TimerWheel) Stop() {
	w.once.Do(func() {
		w.enable.Store(false)
		// close(w.timerQ) // to defend data race warning
		w.ticker.Stop()
	})
}

// Close stops the timer wheel and wait for all grs.
func (w *TimerWheel) Close() {
	w.Stop()
	w.wg.Wait()
}

////////////////////////////////////////////////
// timer
////////////////////////////////////////////////

// TimerType defines a timer task type.
type TimerType int32

const (
	TimerOnce TimerType = 0x1 << 0
	TimerLoop TimerType = 0x1 << 1
)

// AddTimer adds a timer asynchronously and returns a timer struct obj. It returns error if it failed.
//
// Attention that @f may block the timer gr. So u should create a gr to exec ur function asynchronously
// if it may take a long time.
//
// args:
//   @f: timer function.
//   @typ: timer type
//   @period: timer loop interval. its unit is nanosecond.
//   @arg: timer argument which is used by @f.
func (w *TimerWheel) AddTimer(f TimerFunc, typ TimerType, period time.Duration, arg interface{}) (*Timer, error) {
	if !w.enable.Load() {
		return nil, ErrTimeChannelClosed
	}

	t := &Timer{w: w}
	node := newTimerNode(f, typ, int64(period), arg)
	select {
	case w.timerQ <- &timerNodeAction{node: node, action: TimerActionAdd}:
		t.ID = node.ID
		return t, nil
	default:
	}

	return nil, ErrTimeChannelFull
}

func (w *TimerWheel) deleteTimer(t *Timer) error {
	if !w.enable.Load() {
		return ErrTimeChannelClosed
	}

	select {
	case w.timerQ <- &timerNodeAction{action: TimerActionDel, node: &timerNode{ID: t.ID}}:
		return nil
	default:
	}

	return ErrTimeChannelFull
}

func (w *TimerWheel) resetTimer(t *Timer, d time.Duration) error {
	if !w.enable.Load() {
		return ErrTimeChannelClosed
	}

	select {
	case w.timerQ <- &timerNodeAction{action: TimerActionReset, node: &timerNode{ID: t.ID, period: int64(d)}}:
		return nil
	default:
	}

	return ErrTimeChannelFull
}

func sendTime(_ TimerID, t time.Time, arg interface{}) error {
	select {
	case arg.(chan time.Time) <- t:
	default:
		// log.CInfo("sendTime default")
	}

	return nil
}

// NewTimer creates a new Timer that will send
// the current time on its channel after at least duration d.
func (w *TimerWheel) NewTimer(d time.Duration) *Timer {
	c := make(chan time.Time, 1)
	t := &Timer{
		C: c,
	}

	timer, err := w.AddTimer(sendTime, TimerOnce, d, c)
	if err == nil {
		t.ID = timer.ID
		t.w = timer.w
		return t
	}

	close(c)
	return nil
}

// After waits for the duration to elapse and then sends the current time
// on the returned channel.
func (w *TimerWheel) After(d time.Duration) <-chan time.Time {
	//timer := defaultTimer.NewTimer(d)
	//if timer == nil {
	//	return nil
	//}
	//
	//return timer.C
	return w.NewTimer(d).C
}

func goFunc(_ TimerID, _ time.Time, arg interface{}) error {
	go arg.(func())()

	return nil
}

// AfterFunc waits for the duration to elapse and then calls f
// in its own goroutine. It returns a Timer that can
// be used to cancel the call using its Stop method.
func (w *TimerWheel) AfterFunc(d time.Duration, f func()) *Timer {
	t, _ := w.AddTimer(goFunc, TimerOnce, d, f)

	return t
}

// Sleep pauses the current goroutine for at least the duration d.
// A negative or zero duration causes Sleep to return immediately.
func (w *TimerWheel) Sleep(d time.Duration) {
	<-w.NewTimer(d).C
}

////////////////////////////////////////////////
// ticker
////////////////////////////////////////////////

// NewTicker returns a new Ticker containing a channel that will send
// the time on the channel after each tick. The period of the ticks is
// specified by the duration argument. The ticker will adjust the time
// interval or drop ticks to make up for slow receivers.
// The duration d must be greater than zero; if not, NewTicker will
// panic. Stop the ticker to release associated resources.
func (w *TimerWheel) NewTicker(d time.Duration) *Ticker {
	c := make(chan time.Time, 1)

	timer, err := w.AddTimer(sendTime, TimerLoop, d, c)
	if err == nil {
		timer.C = c
		return (*Ticker)(timer)
	}

	close(c)
	return nil
}

// TickFunc returns a Ticker
func (w *TimerWheel) TickFunc(d time.Duration, f func()) *Ticker {
	t, err := w.AddTimer(goFunc, TimerLoop, d, f)
	if err == nil {
		return (*Ticker)(t)
	}

	return nil
}

// Tick is a convenience wrapper for NewTicker providing access to the ticking
// channel only. While Tick is useful for clients that have no need to shut down
// the Ticker, be aware that without a way to shut it down the underlying
// Ticker cannot be recovered by the garbage collector; it "leaks".
// Unlike NewTicker, Tick will return nil if d <= 0.
func (w *TimerWheel) Tick(d time.Duration) <-chan time.Time {
	return w.NewTicker(d).C
}
