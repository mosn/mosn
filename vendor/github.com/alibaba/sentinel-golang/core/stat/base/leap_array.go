package base

import (
	"fmt"
	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
	"runtime"
	"sync/atomic"
	"unsafe"
)

const (
	PtrSize = int(8)
)

// bucketWrap represent a slot to record metrics
// In order to reduce the usage of memory, bucketWrap don't hold length of bucketWrap
// The length of bucketWrap could be seen in leapArray.
// The scope of time is [startTime, startTime+bucketLength)
// The size of bucketWrap is 24(8+16) bytes
type bucketWrap struct {
	// The start timestamp of this statistic bucket wrapper.
	bucketStart uint64
	// The actual data structure to record the metrics (e.g. MetricBucket).
	value atomic.Value
}

func (ww *bucketWrap) resetTo(startTime uint64) {
	ww.bucketStart = startTime
}

func (ww *bucketWrap) isTimeInBucket(now uint64, bucketLengthInMs uint32) bool {
	return ww.bucketStart <= now && now < ww.bucketStart+uint64(bucketLengthInMs)
}

func calculateStartTime(now uint64, bucketLengthInMs uint32) uint64 {
	return now - (now % uint64(bucketLengthInMs))
}

// atomic bucketWrap array to resolve race condition
// atomicBucketWrapArray can not append or delete element after initializing
type atomicBucketWrapArray struct {
	// The base address for real data array
	base unsafe.Pointer
	// The length of slice(array), it can not be modified.
	length int
	data   []*bucketWrap
}

// New atomicBucketWrapArray with initializing field data
// Default, automatically initialize each bucketWrap
// len: length of array
// bucketLengthInMs: bucket length of bucketWrap
// generator: generator to generate bucket
func newAtomicBucketWrapArray(len int, bucketLengthInMs uint32, generator bucketGenerator) *atomicBucketWrapArray {
	ret := &atomicBucketWrapArray{
		length: len,
		data:   make([]*bucketWrap, len),
	}

	// automatically initialize each bucketWrap
	// tail bucketWrap of data is initialized with current time
	startTime := calculateStartTime(util.CurrentTimeMillis(), bucketLengthInMs)
	for i := len - 1; i >= 0; i-- {
		ww := &bucketWrap{
			bucketStart: startTime,
			value:       atomic.Value{},
		}
		ww.value.Store(generator.newEmptyBucket())
		ret.data[i] = ww
		startTime -= uint64(bucketLengthInMs)
	}

	// calculate base address for real data array
	sliHeader := (*util.SliceHeader)(unsafe.Pointer(&ret.data))
	ret.base = unsafe.Pointer((**bucketWrap)(unsafe.Pointer(sliHeader.Data)))
	return ret
}

func (aa *atomicBucketWrapArray) elementOffset(idx int) unsafe.Pointer {
	if idx >= aa.length && idx < 0 {
		panic(fmt.Sprintf("The index (%d) is out of bounds, length is %d.", idx, aa.length))
	}
	basePtr := aa.base
	return unsafe.Pointer(uintptr(basePtr) + uintptr(idx*PtrSize))
}

func (aa *atomicBucketWrapArray) get(idx int) *bucketWrap {
	// aa.elementOffset(idx) return the secondary pointer of bucketWrap, which is the pointer to the aa.data[idx]
	// then convert to (*unsafe.Pointer)
	return (*bucketWrap)(atomic.LoadPointer((*unsafe.Pointer)(aa.elementOffset(idx))))
}

func (aa *atomicBucketWrapArray) compareAndSet(idx int, except, update *bucketWrap) bool {
	// aa.elementOffset(idx) return the secondary pointer of bucketWrap, which is the pointer to the aa.data[idx]
	// then convert to (*unsafe.Pointer)
	// update secondary pointer
	return atomic.CompareAndSwapPointer((*unsafe.Pointer)(aa.elementOffset(idx)), unsafe.Pointer(except), unsafe.Pointer(update))
}

// The bucketWrap leap array,
// sampleCount represent the number of bucketWrap
// intervalInMs represent the interval of leapArray.
// For example, bucketLengthInMs is 500ms, intervalInMs is 1min, so sampleCount is 120.
type leapArray struct {
	bucketLengthInMs uint32
	sampleCount      uint32
	intervalInMs     uint32
	array            *atomicBucketWrapArray
	// update lock
	updateLock mutex
}

func (la *leapArray) currentBucket(bg bucketGenerator) (*bucketWrap, error) {
	return la.currentBucketOfTime(util.CurrentTimeMillis(), bg)
}

func (la *leapArray) currentBucketOfTime(now uint64, bg bucketGenerator) (*bucketWrap, error) {
	if now < 0 {
		return nil, errors.New("Current time is less than 0.")
	}

	idx := la.calculateTimeIdx(now)
	bucketStart := calculateStartTime(now, la.bucketLengthInMs)

	for { //spin to get the current bucketWrap
		old := la.array.get(idx)
		if old == nil {
			// because la.array.data had initiated when new la.array
			// theoretically, here is not reachable
			newWrap := &bucketWrap{
				bucketStart: bucketStart,
				value:       atomic.Value{},
			}
			newWrap.value.Store(bg.newEmptyBucket())
			if la.array.compareAndSet(idx, nil, newWrap) {
				return newWrap, nil
			} else {
				runtime.Gosched()
			}
		} else if bucketStart == atomic.LoadUint64(&old.bucketStart) {
			return old, nil
		} else if bucketStart > atomic.LoadUint64(&old.bucketStart) {
			// current time has been next cycle of leapArray and leapArray dont't count in last cycle.
			// reset bucketWrap
			if la.updateLock.TryLock() {
				old = bg.resetBucketTo(old, bucketStart)
				la.updateLock.Unlock()
				return old, nil
			} else {
				runtime.Gosched()
			}
		} else if bucketStart < old.bucketStart {
			// TODO: reserve for some special case (e.g. when occupying "future" buckets).
			return nil, errors.New(fmt.Sprintf("Provided time timeMillis=%d is already behind old.bucketStart=%d.", bucketStart, old.bucketStart))
		}
	}
}

func (la *leapArray) calculateTimeIdx(now uint64) int {
	timeId := now / uint64(la.bucketLengthInMs)
	return int(timeId) % la.array.length
}

//  Get all bucketWrap between [current time -1000ms, current time]
func (la *leapArray) values() []*bucketWrap {
	return la.valuesWithTime(util.CurrentTimeMillis())
}

func (la *leapArray) valuesWithTime(now uint64) []*bucketWrap {
	if now <= 0 {
		return make([]*bucketWrap, 0)
	}
	ret := make([]*bucketWrap, 0)
	for i := 0; i < la.array.length; i++ {
		ww := la.array.get(i)
		if ww == nil || la.isBucketDeprecated(now, ww) {
			continue
		}
		ret = append(ret, ww)
	}
	return ret
}

func (la *leapArray) ValuesConditional(now uint64, predicate base.TimePredicate) []*bucketWrap {
	if now <= 0 {
		return make([]*bucketWrap, 0)
	}
	ret := make([]*bucketWrap, 0)
	for i := 0; i < la.array.length; i++ {
		ww := la.array.get(i)
		if ww == nil || la.isBucketDeprecated(now, ww) || !predicate(atomic.LoadUint64(&ww.bucketStart)) {
			continue
		}
		ret = append(ret, ww)
	}
	return ret
}

// Judge whether the bucketWrap is expired
func (la *leapArray) isBucketDeprecated(now uint64, ww *bucketWrap) bool {
	ws := atomic.LoadUint64(&ww.bucketStart)
	return (now - ws) > uint64(la.intervalInMs)
}

// Generic interface to generate bucket
type bucketGenerator interface {
	// called when timestamp entry a new slot interval
	newEmptyBucket() interface{}

	// reset the bucketWrap, clear all data of bucketWrap
	resetBucketTo(ww *bucketWrap, startTime uint64) *bucketWrap
}
