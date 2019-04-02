package shm

import (
	"github.com/alipay/sofa-mosn/pkg/shm"
	"os"
	"sync/atomic"
	"time"
	"runtime"
	"unsafe"
	"errors"
	"github.com/alipay/sofa-mosn/pkg/server/keeper"
	"log"
)

var (
	pageSize = 4 * 1024
	pid      = uint32(os.Getpid())

	defaultZone *zone
)

// InitDefaultMetricsZone used to initialize the default zone according to the configuration.
// And the default zone will detach while process exiting
func InitDefaultMetricsZone(name string, size int) {
	zone := createMetricsZone(name, size)

	defaultZone = zone

	keeper.OnProcessShutDown(func() error {
		zone.Detach()
		return nil
	})
}

// InitMetricsZone used to initialize the default zone according to the configuration.
// It's caller's responsibility to detach the zone.
func InitMetricsZone(name string, size int) *zone {
	defaultZone = createMetricsZone(name, size)
	return defaultZone
}

// createMetricsZone used to create new shm-based metrics zone. It's caller's responsibility
// to detach the zone.
func createMetricsZone(name string, size int) *zone {
	zone, err := newSharedMetrics(name, size)
	if err != nil {
		log.Fatalln("open shared memory for metrics failed:", err)
	}
	return zone
}

// zone is the in-heap struct that holds the reference to the entire metrics shared memory.
// ATTENTION: entries is modified so that it points to the shared memory entries address.
type zone struct {
	span *shm.ShmSpan

	mutex *uint32
	ref   *uint32

	set *hashSet // mutex + ref = 64bit, so atomic ops has no problem
}

func newSharedMetrics(name string, size int) (*zone, error) {
	alignedSize := align(size, pageSize)

	span, err := shm.Alloc(name, alignedSize)
	if err != nil {
		return nil, err
	}
	// 1. mutex and ref
	mutex, err := span.Alloc(4)
	if err != nil {
		return nil, err
	}

	ref, err := span.Alloc(4)
	if err != nil {
		return nil, err
	}

	// 2. calc hashSet size

	// assuming that 100 entries with 50 slots, so the ratio of occupied memory is
	// entries:slots  = 100 x 128 : 50 x 4 = 64 : 1
	// so assuming slots memory size is N, total allocated memory size is M, then we have:
	// M - 1024 < 65N + 28 <= M

	slotsNum := (alignedSize - 28) / (65 * 4)
	slotsSize := slotsNum * 4
	entryNum := slotsNum * 2
	entrySize := slotsSize * 64

	hashSegSize := entrySize + 20 + slotsSize
	hashSegment, err := span.Alloc(hashSegSize)
	if err != nil {
		return nil, err
	}

	set, err := newHashSet(hashSegment, hashSegSize, entryNum, slotsNum)
	if err != nil {
		return nil, err
	}

	zone := &zone{
		span:  span,
		set:   set,
		mutex: (*uint32)(unsafe.Pointer(mutex)),
		ref:   (*uint32)(unsafe.Pointer(ref)),
	}
	// add ref
	atomic.AddUint32(zone.ref, 1)

	return zone, nil
}

func (z *zone) lock() {
	times := 0

	// 5ms spin interval, 5 times burst
	for true {
		if atomic.CompareAndSwapUint32(z.mutex, 0, pid) {
			return
		}

		time.Sleep(time.Millisecond * 5)
		times++

		if times%5 == 0 {
			runtime.Gosched()
		}
	}
}

func (z *zone) unlock() {
	times := 0

	// 5ms spin interval, 5 times burst
	for true {
		if atomic.CompareAndSwapUint32(z.mutex, pid, 0) {
			return
		}

		time.Sleep(time.Millisecond * 5)
		times++

		if times%5 == 0 {
			runtime.Gosched()
		}
	}
}

func (z *zone) alloc(name string) (*hashEntry, error) {
	z.lock()
	defer z.unlock()

	entry, create := z.set.Alloc(name)
	if entry == nil {
		// TODO log & stat
		return nil, errors.New("alloc failed")
	}

	// for existed entry, increase its reference
	if !create {
		entry.incRef()
	}

	return entry, nil
}

func (z *zone) free(entry *hashEntry) error {
	z.lock()
	defer z.unlock()

	z.set.Free(entry)
	return nil
}

func (z *zone) Detach() {
	// ensure all process detached
	if atomic.AddUint32(z.ref, ^uint32(0)) == 0 {
		shm.DeAlloc(z.span)
	}
}

func align(size, alignment int) int {
	return (size + alignment - 1) & ^(alignment - 1);
}
