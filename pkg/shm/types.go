package shm

import (
	"errors"
	"unsafe"
	"github.com/alipay/sofa-mosn/pkg/types"
	"os"
	"fmt"
)

var (
	errNotEnough = errors.New("span capacity is not enough")
)

func path(name string) string {
	return types.MosnConfigPath + string(os.PathSeparator) + fmt.Sprintf("mosn_shm_%s", name)
}

// TODO support process level lock use cas
type ShmSpan struct {
	origin []byte
	name   string

	data   uintptr
	offset int
	size   int
}

func NewShmSpan(name string, data []byte) *ShmSpan {
	return &ShmSpan{
		name:   name,
		origin: data,
		data:   uintptr(unsafe.Pointer(&data[0])),
		size:   len(data),
	}
}

func (s *ShmSpan) Alloc(size int) (uintptr, error) {
	if s.offset+size > s.size {
		return 0, errNotEnough
	}

	ptr := s.data + uintptr(s.offset)
	s.offset += size
	return ptr, nil
}

func (s *ShmSpan) Data() uintptr {
	return s.data
}

func (s *ShmSpan) Origin() []byte {
	return s.origin
}
