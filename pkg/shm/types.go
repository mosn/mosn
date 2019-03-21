package shm

import (
	"errors"
	"sync"
	"unsafe"
)

const defaultCachelineSize = 128

var (
	totalSpanCount uint32
	errNotEnough   = errors.New("span capacity is not enough")
)

type ShmSpan struct {
	sync.Mutex
	origin []byte

	data   uintptr
	offset int
	size   int
}

func NewShmSpan(data []byte) *ShmSpan {
	return &ShmSpan{
		origin: data,
		data:   uintptr(unsafe.Pointer(&data[0])),
		size:   len(data),
	}
}

func (s *ShmSpan) Alloc(size int) (uintptr, error) {
	s.Lock()
	defer s.Unlock()

	if s.offset+size >= s.size {
		return 0, errNotEnough
	}

	ptr := s.data + uintptr(s.offset)
	s.offset += size
	return ptr, nil
}
