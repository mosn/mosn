// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package circuitbreaker

import (
	"reflect"
	"sync/atomic"

	"github.com/alibaba/sentinel-golang/core/base"
	sbase "github.com/alibaba/sentinel-golang/core/stat/base"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

//
//  Circuit Breaker State Machine:
//
//                                 switch to open based on rule
//				+-----------------------------------------------------------------------+
//				|                                                                       |
//				|                                                                       v
//		+----------------+                   +----------------+      Probe      +----------------+
//		|                |                   |                |<----------------|                |
//		|                |   Probe succeed   |                |                 |                |
//		|     Closed     |<------------------|    HalfOpen    |                 |      Open      |
//		|                |                   |                |   Probe failed  |                |
//		|                |                   |                +---------------->|                |
//		+----------------+                   +----------------+                 +----------------+
type State int32

const (
	Closed State = iota
	HalfOpen
	Open
)

func newState() *State {
	var state State
	state = Closed

	return &state
}

func (s *State) String() string {
	switch s.get() {
	case Closed:
		return "Closed"
	case HalfOpen:
		return "HalfOpen"
	case Open:
		return "Open"
	default:
		return "Undefined"
	}
}

func (s *State) get() State {
	return State(atomic.LoadInt32((*int32)(s)))
}

func (s *State) set(update State) {
	atomic.StoreInt32((*int32)(s), int32(update))
}

func (s *State) cas(expect State, update State) bool {
	return atomic.CompareAndSwapInt32((*int32)(s), int32(expect), int32(update))
}

// StateChangeListener listens on the circuit breaker state change event
type StateChangeListener interface {
	// OnTransformToClosed is triggered when circuit breaker state transformed to Closed.
	// Argument rule is copy from circuit breaker's rule, any changes of rule don't take effect for circuit breaker
	// Copying rule has a performance penalty and avoids invalid listeners as much as possible
	OnTransformToClosed(prev State, rule Rule)

	// OnTransformToOpen is triggered when circuit breaker state transformed to Open.
	// The "snapshot" indicates the triggered value when the transformation occurs.
	// Argument rule is copy from circuit breaker's rule, any changes of rule don't take effect for circuit breaker
	// Copying rule has a performance penalty and avoids invalid listeners as much as possible
	OnTransformToOpen(prev State, rule Rule, snapshot interface{})

	// OnTransformToHalfOpen is triggered when circuit breaker state transformed to HalfOpen.
	// Argument rule is copy from circuit breaker's rule, any changes of rule don't take effect for circuit breaker
	// Copying rule has a performance penalty and avoids invalid listeners as much as possible
	OnTransformToHalfOpen(prev State, rule Rule)
}

// CircuitBreaker is the basic interface of circuit breaker
type CircuitBreaker interface {
	// BoundRule returns the associated circuit breaking rule.
	BoundRule() *Rule
	// BoundStat returns the associated statistic data structure.
	BoundStat() interface{}
	// TryPass acquires permission of an invocation only if it is available at the time of invocation.
	TryPass(ctx *base.EntryContext) bool
	// CurrentState returns current state of the circuit breaker.
	CurrentState() State
	// OnRequestComplete record a completed request with the given response time as well as error (if present),
	// and handle state transformation of the circuit breaker.
	// OnRequestComplete is called only when a passed invocation finished.
	OnRequestComplete(rtt uint64, err error)
}

//================================= circuitBreakerBase ====================================
// circuitBreakerBase encompasses the common fields of circuit breaker.
type circuitBreakerBase struct {
	rule *Rule
	// retryTimeoutMs represents recovery timeout (in milliseconds) before the circuit breaker opens.
	// During the open period, no requests are permitted until the timeout has elapsed.
	// After that, the circuit breaker will transform to half-open state for trying a few "trial" requests.
	retryTimeoutMs uint32
	// nextRetryTimestampMs is the time circuit breaker could probe
	nextRetryTimestampMs uint64
	// state is the state machine of circuit breaker
	state *State
}

func (b *circuitBreakerBase) BoundRule() *Rule {
	return b.rule
}

func (b *circuitBreakerBase) CurrentState() State {
	return b.state.get()
}

func (b *circuitBreakerBase) retryTimeoutArrived() bool {
	return util.CurrentTimeMillis() >= atomic.LoadUint64(&b.nextRetryTimestampMs)
}

func (b *circuitBreakerBase) updateNextRetryTimestamp() {
	atomic.StoreUint64(&b.nextRetryTimestampMs, util.CurrentTimeMillis()+uint64(b.retryTimeoutMs))
}

// fromClosedToOpen updates circuit breaker state machine from closed to open.
// Return true only if current goroutine successfully accomplished the transformation.
func (b *circuitBreakerBase) fromClosedToOpen(snapshot interface{}) bool {
	if b.state.cas(Closed, Open) {
		b.updateNextRetryTimestamp()
		for _, listener := range stateChangeListeners {
			listener.OnTransformToOpen(Closed, *b.rule, snapshot)
		}
		return true
	}
	return false
}

// fromOpenToHalfOpen updates circuit breaker state machine from open to half-open.
// Return true only if current goroutine successfully accomplished the transformation.
func (b *circuitBreakerBase) fromOpenToHalfOpen(ctx *base.EntryContext) bool {
	if b.state.cas(Open, HalfOpen) {
		for _, listener := range stateChangeListeners {
			listener.OnTransformToHalfOpen(Open, *b.rule)
		}

		entry := ctx.Entry()
		if entry == nil {
			logging.Error(errors.New("nil entry"), "Nil entry in circuitBreakerBase.fromOpenToHalfOpen()", "rule", b.rule)
		} else {
			// add hook for entry exit
			// if the current circuit breaker performs the probe through this entry, but the entry was blocked,
			// this hook will guarantee current circuit breaker state machine will rollback to Open from Half-Open
			entry.WhenExit(func(entry *base.SentinelEntry, ctx *base.EntryContext) error {
				if ctx.IsBlocked() && b.state.cas(HalfOpen, Open) {
					for _, listener := range stateChangeListeners {
						listener.OnTransformToOpen(HalfOpen, *b.rule, 1.0)
					}
				}
				return nil
			})
		}
		return true
	}
	return false
}

// fromHalfOpenToOpen updates circuit breaker state machine from half-open to open.
// Return true only if current goroutine successfully accomplished the transformation.
func (b *circuitBreakerBase) fromHalfOpenToOpen(snapshot interface{}) bool {
	if b.state.cas(HalfOpen, Open) {
		b.updateNextRetryTimestamp()
		for _, listener := range stateChangeListeners {
			listener.OnTransformToOpen(HalfOpen, *b.rule, snapshot)
		}
		return true
	}
	return false
}

// fromHalfOpenToOpen updates circuit breaker state machine from half-open to closed
// Return true only if current goroutine successfully accomplished the transformation.
func (b *circuitBreakerBase) fromHalfOpenToClosed() bool {
	if b.state.cas(HalfOpen, Closed) {
		for _, listener := range stateChangeListeners {
			listener.OnTransformToClosed(HalfOpen, *b.rule)
		}
		return true
	}
	return false
}

//================================= slowRtCircuitBreaker ====================================
type slowRtCircuitBreaker struct {
	circuitBreakerBase
	stat                *slowRequestLeapArray
	maxAllowedRt        uint64
	maxSlowRequestRatio float64
	minRequestAmount    uint64
}

func newSlowRtCircuitBreakerWithStat(r *Rule, stat *slowRequestLeapArray) *slowRtCircuitBreaker {
	return &slowRtCircuitBreaker{
		circuitBreakerBase: circuitBreakerBase{
			rule:                 r,
			retryTimeoutMs:       r.RetryTimeoutMs,
			nextRetryTimestampMs: 0,
			state:                newState(),
		},
		stat:                stat,
		maxAllowedRt:        r.MaxAllowedRtMs,
		maxSlowRequestRatio: r.Threshold,
		minRequestAmount:    r.MinRequestAmount,
	}
}

func newSlowRtCircuitBreaker(r *Rule) (*slowRtCircuitBreaker, error) {
	interval := r.StatIntervalMs
	bucketCount := getRuleStatSlidingWindowBucketCount(r)
	stat := &slowRequestLeapArray{}
	leapArray, err := sbase.NewLeapArray(bucketCount, interval, stat)
	if err != nil {
		return nil, err
	}
	stat.data = leapArray

	return newSlowRtCircuitBreakerWithStat(r, stat), nil
}

func (b *slowRtCircuitBreaker) BoundStat() interface{} {
	return b.stat
}

// TryPass checks circuit breaker based on state machine of circuit breaker.
func (b *slowRtCircuitBreaker) TryPass(ctx *base.EntryContext) bool {
	curStatus := b.CurrentState()
	if curStatus == Closed {
		return true
	} else if curStatus == Open {
		// switch state to half-open to probe if retry timeout
		if b.retryTimeoutArrived() && b.fromOpenToHalfOpen(ctx) {
			return true
		}
	}
	return false
}

func (b *slowRtCircuitBreaker) OnRequestComplete(rt uint64, _ error) {
	// add slow and add total
	metricStat := b.stat
	counter, curErr := metricStat.currentCounter()
	if curErr != nil {
		logging.Error(curErr, "Fail to get current counter in slowRtCircuitBreaker#OnRequestComplete().",
			"rule", b.rule)
		return
	}
	if rt > b.maxAllowedRt {
		atomic.AddUint64(&counter.slowCount, 1)
	}
	atomic.AddUint64(&counter.totalCount, 1)

	slowCount := uint64(0)
	totalCount := uint64(0)
	counters := metricStat.allCounter()
	for _, c := range counters {
		slowCount += atomic.LoadUint64(&c.slowCount)
		totalCount += atomic.LoadUint64(&c.totalCount)
	}
	slowRatio := float64(slowCount) / float64(totalCount)

	// handleStateChange
	curStatus := b.CurrentState()
	if curStatus == Open {
		return
	} else if curStatus == HalfOpen {
		if rt > b.maxAllowedRt {
			// fail to probe
			b.fromHalfOpenToOpen(1.0)
		} else {
			// succeed to probe
			b.fromHalfOpenToClosed()
			b.resetMetric()
		}
		return
	}

	// current state is CLOSED
	if totalCount < b.minRequestAmount {
		return
	}

	if slowRatio > b.maxSlowRequestRatio || util.Float64Equals(slowRatio, b.maxSlowRequestRatio) {
		curStatus = b.CurrentState()
		switch curStatus {
		case Closed:
			b.fromClosedToOpen(slowRatio)
		case HalfOpen:
			b.fromHalfOpenToOpen(slowRatio)
		default:
		}
	}
	return
}

func (b *slowRtCircuitBreaker) resetMetric() {
	for _, c := range b.stat.allCounter() {
		c.reset()
	}
}

type slowRequestCounter struct {
	slowCount  uint64
	totalCount uint64
}

func (c *slowRequestCounter) reset() {
	atomic.StoreUint64(&c.slowCount, 0)
	atomic.StoreUint64(&c.totalCount, 0)
}

type slowRequestLeapArray struct {
	data *sbase.LeapArray
}

func (s *slowRequestLeapArray) NewEmptyBucket() interface{} {
	return &slowRequestCounter{
		slowCount:  0,
		totalCount: 0,
	}
}

func (s *slowRequestLeapArray) ResetBucketTo(bw *sbase.BucketWrap, startTime uint64) *sbase.BucketWrap {
	atomic.StoreUint64(&bw.BucketStart, startTime)
	bw.Value.Store(&slowRequestCounter{
		slowCount:  0,
		totalCount: 0,
	})
	return bw
}

func (s *slowRequestLeapArray) currentCounter() (*slowRequestCounter, error) {
	curBucket, err := s.data.CurrentBucket(s)
	if err != nil {
		return nil, err
	}
	if curBucket == nil {
		return nil, errors.New("nil BucketWrap")
	}
	mb := curBucket.Value.Load()
	if mb == nil {
		return nil, errors.New("nil slowRequestCounter")
	}
	counter, ok := mb.(*slowRequestCounter)
	if !ok {
		return nil, errors.Errorf("bucket fail to do type assert, expect: *slowRequestCounter, in fact: %s", reflect.TypeOf(mb).Name())
	}
	return counter, nil
}

func (s *slowRequestLeapArray) allCounter() []*slowRequestCounter {
	buckets := s.data.Values()
	ret := make([]*slowRequestCounter, 0, len(buckets))
	for _, b := range buckets {
		mb := b.Value.Load()
		if mb == nil {
			logging.Error(errors.New("current bucket atomic Value is nil"), "Current bucket atomic Value is nil in slowRequestLeapArray.allCounter()")
			continue
		}
		counter, ok := mb.(*slowRequestCounter)
		if !ok {
			logging.Error(errors.New("bucket data type error"), "Bucket data type error in slowRequestLeapArray.allCounter()", "expect type", "*slowRequestCounter", "actual type", reflect.TypeOf(mb).Name())
			continue
		}
		ret = append(ret, counter)
	}
	return ret
}

//================================= errorRatioCircuitBreaker ====================================
type errorRatioCircuitBreaker struct {
	circuitBreakerBase
	minRequestAmount    uint64
	errorRatioThreshold float64

	stat *errorCounterLeapArray
}

func newErrorRatioCircuitBreakerWithStat(r *Rule, stat *errorCounterLeapArray) *errorRatioCircuitBreaker {
	return &errorRatioCircuitBreaker{
		circuitBreakerBase: circuitBreakerBase{
			rule:                 r,
			retryTimeoutMs:       r.RetryTimeoutMs,
			nextRetryTimestampMs: 0,
			state:                newState(),
		},
		minRequestAmount:    r.MinRequestAmount,
		errorRatioThreshold: r.Threshold,
		stat:                stat,
	}
}

func newErrorRatioCircuitBreaker(r *Rule) (*errorRatioCircuitBreaker, error) {
	interval := r.StatIntervalMs
	bucketCount := getRuleStatSlidingWindowBucketCount(r)
	stat := &errorCounterLeapArray{}
	leapArray, err := sbase.NewLeapArray(bucketCount, interval, stat)
	if err != nil {
		return nil, err
	}
	stat.data = leapArray
	return newErrorRatioCircuitBreakerWithStat(r, stat), nil
}

func (b *errorRatioCircuitBreaker) BoundStat() interface{} {
	return b.stat
}

func (b *errorRatioCircuitBreaker) TryPass(ctx *base.EntryContext) bool {
	curStatus := b.CurrentState()
	if curStatus == Closed {
		return true
	} else if curStatus == Open {
		// switch state to half-open to probe if retry timeout
		if b.retryTimeoutArrived() && b.fromOpenToHalfOpen(ctx) {
			return true
		}
	}
	return false
}

func (b *errorRatioCircuitBreaker) OnRequestComplete(_ uint64, err error) {
	metricStat := b.stat
	counter, curErr := metricStat.currentCounter()
	if curErr != nil {
		logging.Error(curErr, "Fail to get current counter in errorRatioCircuitBreaker#OnRequestComplete().",
			"rule", b.rule)
		return
	}
	if err != nil {
		atomic.AddUint64(&counter.errorCount, 1)
	}
	atomic.AddUint64(&counter.totalCount, 1)

	errorCount := uint64(0)
	totalCount := uint64(0)
	counters := metricStat.allCounter()
	for _, c := range counters {
		errorCount += atomic.LoadUint64(&c.errorCount)
		totalCount += atomic.LoadUint64(&c.totalCount)
	}
	errorRatio := float64(errorCount) / float64(totalCount)

	// handleStateChangeWhenThresholdExceeded
	curStatus := b.CurrentState()
	if curStatus == Open {
		return
	}
	if curStatus == HalfOpen {
		if err == nil {
			b.fromHalfOpenToClosed()
			b.resetMetric()
		} else {
			b.fromHalfOpenToOpen(1.0)
		}
		return
	}

	// current state is CLOSED
	if totalCount < b.minRequestAmount {
		return
	}
	if errorRatio > b.errorRatioThreshold || util.Float64Equals(errorRatio, b.errorRatioThreshold) {
		curStatus = b.CurrentState()
		switch curStatus {
		case Closed:
			b.fromClosedToOpen(errorRatio)
		case HalfOpen:
			b.fromHalfOpenToOpen(errorRatio)
		default:
		}
	}
}

func (b *errorRatioCircuitBreaker) resetMetric() {
	for _, c := range b.stat.allCounter() {
		c.reset()
	}
}

type errorCounter struct {
	errorCount uint64
	totalCount uint64
}

func (c *errorCounter) reset() {
	atomic.StoreUint64(&c.errorCount, 0)
	atomic.StoreUint64(&c.totalCount, 0)
}

type errorCounterLeapArray struct {
	data *sbase.LeapArray
}

func (s *errorCounterLeapArray) NewEmptyBucket() interface{} {
	return &errorCounter{
		errorCount: 0,
		totalCount: 0,
	}
}

func (s *errorCounterLeapArray) ResetBucketTo(bw *sbase.BucketWrap, startTime uint64) *sbase.BucketWrap {
	atomic.StoreUint64(&bw.BucketStart, startTime)
	bw.Value.Store(&errorCounter{
		errorCount: 0,
		totalCount: 0,
	})
	return bw
}

func (s *errorCounterLeapArray) currentCounter() (*errorCounter, error) {
	curBucket, err := s.data.CurrentBucket(s)
	if err != nil {
		return nil, err
	}
	if curBucket == nil {
		return nil, errors.New("nil BucketWrap")
	}
	mb := curBucket.Value.Load()
	if mb == nil {
		return nil, errors.New("nil errorCounter")
	}
	counter, ok := mb.(*errorCounter)
	if !ok {
		return nil, errors.Errorf("bucket fail to do type assert, expect: *errorCounter, in fact: %s", reflect.TypeOf(mb).Name())
	}
	return counter, nil
}

func (s *errorCounterLeapArray) allCounter() []*errorCounter {
	buckets := s.data.Values()
	ret := make([]*errorCounter, 0, len(buckets))
	for _, b := range buckets {
		mb := b.Value.Load()
		if mb == nil {
			logging.Error(errors.New("current bucket atomic Value is nil"), "Current bucket atomic Value is nil in errorCounterLeapArray.allCounter()")
			continue
		}
		counter, ok := mb.(*errorCounter)
		if !ok {
			logging.Error(errors.New("bucket data type error"), "Bucket data type error in errorCounterLeapArray.allCounter()", "expect type", "*errorCounter", "actual type", reflect.TypeOf(mb).Name())
			continue
		}
		ret = append(ret, counter)
	}
	return ret
}

//================================= errorCountCircuitBreaker ====================================
type errorCountCircuitBreaker struct {
	circuitBreakerBase
	minRequestAmount    uint64
	errorCountThreshold uint64

	stat *errorCounterLeapArray
}

func newErrorCountCircuitBreakerWithStat(r *Rule, stat *errorCounterLeapArray) *errorCountCircuitBreaker {
	return &errorCountCircuitBreaker{
		circuitBreakerBase: circuitBreakerBase{
			rule:                 r,
			retryTimeoutMs:       r.RetryTimeoutMs,
			nextRetryTimestampMs: 0,
			state:                newState(),
		},
		minRequestAmount:    r.MinRequestAmount,
		errorCountThreshold: uint64(r.Threshold),
		stat:                stat,
	}
}

func newErrorCountCircuitBreaker(r *Rule) (*errorCountCircuitBreaker, error) {
	interval := r.StatIntervalMs
	bucketCount := getRuleStatSlidingWindowBucketCount(r)
	stat := &errorCounterLeapArray{}
	leapArray, err := sbase.NewLeapArray(bucketCount, interval, stat)
	if err != nil {
		return nil, err
	}
	stat.data = leapArray
	return newErrorCountCircuitBreakerWithStat(r, stat), nil
}

func (b *errorCountCircuitBreaker) BoundStat() interface{} {
	return b.stat
}

func (b *errorCountCircuitBreaker) TryPass(ctx *base.EntryContext) bool {
	curStatus := b.CurrentState()
	if curStatus == Closed {
		return true
	} else if curStatus == Open {
		// switch state to half-open to probe if retry timeout
		if b.retryTimeoutArrived() && b.fromOpenToHalfOpen(ctx) {
			return true
		}
	}
	return false
}

func (b *errorCountCircuitBreaker) OnRequestComplete(_ uint64, err error) {
	metricStat := b.stat
	counter, curErr := metricStat.currentCounter()
	if curErr != nil {
		logging.Error(curErr, "Fail to get current counter in errorCountCircuitBreaker#OnRequestComplete().",
			"rule", b.rule)
		return
	}
	if err != nil {
		atomic.AddUint64(&counter.errorCount, 1)
	}
	atomic.AddUint64(&counter.totalCount, 1)

	errorCount := uint64(0)
	totalCount := uint64(0)
	counters := metricStat.allCounter()
	for _, c := range counters {
		errorCount += atomic.LoadUint64(&c.errorCount)
		totalCount += atomic.LoadUint64(&c.totalCount)
	}
	// handleStateChangeWhenThresholdExceeded
	curStatus := b.CurrentState()
	if curStatus == Open {
		return
	}
	if curStatus == HalfOpen {
		if err == nil {
			b.fromHalfOpenToClosed()
			b.resetMetric()
		} else {
			b.fromHalfOpenToOpen(1)
		}
		return
	}
	// current state is CLOSED
	if totalCount < b.minRequestAmount {
		return
	}
	if errorCount >= b.errorCountThreshold {
		curStatus = b.CurrentState()
		switch curStatus {
		case Closed:
			b.fromClosedToOpen(errorCount)
		case HalfOpen:
			b.fromHalfOpenToOpen(errorCount)
		default:
		}
	}
}

func (b *errorCountCircuitBreaker) resetMetric() {
	for _, c := range b.stat.allCounter() {
		c.reset()
	}
}
