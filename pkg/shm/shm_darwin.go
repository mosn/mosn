package shm

import (
	"os"
	"syscall"
	"sync/atomic"
	"fmt"
)

// TODO name confirm
var persistStorePrefix = os.Getenv("persist_store_prefix")

func Alloc(size int) (*ShmSpan, error) {
	index := atomic.AddUint32(&totalSpanCount, 1)

	f, err := os.OpenFile(persistStorePrefix+ fmt.Sprintf("mosn_mmap_%d", index), os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	// extend file
	if err := f.Truncate(int64(size)); err != nil {
		return nil, err
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_WRITE, syscall.MAP_SHARED)

	if err != nil {
		return nil, err
	}

	return NewShmSpan(data), nil
}

func DeAlloc(span *ShmSpan) error {
	return syscall.Munmap(span.origin)
}
