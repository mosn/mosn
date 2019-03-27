package shm

import (
	"github.com/alipay/sofa-mosn/pkg/shm"
	"unsafe"
	"reflect"
	"sync/atomic"
	"os"
	"sync"
	"errors"
)

var (
	metadataSize = int(unsafe.Sizeof(metadata{}))
	pid          = uint32(os.Getpid())
)
// zone is the in-heap struct that holds the reference to the entire metrics shared memory.
// ATTENTION: entries is modified so that it points to the shared memory entries address.
type zone struct {
	*metadata
	entries []metricsEntry

	span *shm.ShmSpan

	indexMux sync.RWMutex
	index    map[string]int

	freeList []int
}

// metricsEntry is the mapping for metrics metadata memory-layout in shared memory.
//
// This struct should never be instantiated.
// TODO move logic to shm package
type metadata struct {
	lock uint32 // 4
	size uint32 // 4
	used uint32 // 4
	ref  uint32 //4

	padding [112]byte
}

func NewSharedMetrics(name string, size int) (MetricsZone, error) {
	span, err := shm.Alloc(name, size)
	if err != nil {
		return nil, err
	}

	zone := &zone{
		span:  span,
		index: make(map[string]int),
	}

	// 1. alloc metadata
	metadataPtr, err := span.Alloc(metadataSize)
	if err != nil {
		return nil, err
	}

	// 2. alloc entries
	entryMemSize := size - metadataSize
	entries, err := span.Alloc(entryMemSize)
	if err != nil {
		return nil, err
	}

	length := entryMemSize / entrySize

	entrySlice := (*reflect.SliceHeader)(unsafe.Pointer(&zone.entries))
	entrySlice.Data = entries
	entrySlice.Len = length
	entrySlice.Cap = length

	zone.metadata = (*metadata)(unsafe.Pointer(metadataPtr))
	zone.size = uint32(length)
	zone.used = 0
	atomic.AddUint32(&zone.ref, 1)

	return zone, nil
}

func (z *zone) tryLock(timeoutMS int) bool {
	// 10ms interval
	for t := 0; t < timeoutMS; t += 10 {
		if atomic.CompareAndSwapUint32(&z.metadata.lock, 0, pid) {
			return true
		}
	}
	return false
}

func (z *zone) unlock() bool {
	return atomic.CompareAndSwapUint32(&z.metadata.lock, pid, 0)
}

func (z *zone) GetEntry(name string) (*metricsEntry, error) {
	// 1. search in local cache
	z.indexMux.RLock()
	offset, ok := z.index[name]
	z.indexMux.RUnlock()

	if ok {
		entry := &z.entries[offset]
		entry.incRef()
		return entry, nil
	}

	// 2. search in shm entry list
	// TODO use block memory hash set for performance
	nameBytes := []byte(name)
	for i := 0; i < int(z.metadata.used); i++ {
		entry := &z.entries[i]
		if entry.equalName(nameBytes) {
			return entry, nil
		}
	}
	return nil, nil
}

func (z *zone) AllocEntry(name string) (*metricsEntry, error) {
	// 1. search in local cache
	z.indexMux.RLock()
	offset, ok := z.index[name]
	z.indexMux.RUnlock()

	if ok {
		entry := &z.entries[offset]
		entry.incRef()
		return entry, nil
	}

	// 2. search in shm entry list
	// TODO use block memory hash set for performance
	nameBytes := []byte(name)
	for i := 0; i < int(z.used); i++ {
		entry := &z.entries[i]
		if entry.equalName(nameBytes) {
			entry.incRef()
			return entry, nil
		}
	}

	// 3.1 TODO alloc from Free list
	// 3.2 alloc new one
	if z.used >= z.size {
		return nil, errors.New("capacity not enough")
	}

	entry := &z.entries[z.used]
	entry.assignName([]byte(name))
	entry.ref = 1

	z.indexMux.Lock()
	z.index[name] = int(z.metadata.used)
	z.indexMux.Unlock()

	z.metadata.used++

	return entry, nil
}

func (z *zone) Free() {
	// ensure all process detached
	if atomic.AddUint32(&z.ref, ^uint32(0)) == 0 {
		shm.DeAlloc(z.span)
	}
}

func (z *zone) FreeEntry(entry *metricsEntry) {
	if atomic.AddUint32(&entry.ref, ^uint32(0)) == 0 {
		// TODO add to free list
	}
}
