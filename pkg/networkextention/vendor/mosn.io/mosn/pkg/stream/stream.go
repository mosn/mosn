package stream

import (
	"sync"
	"sync/atomic"

	"mosn.io/mosn/pkg/types"
)

const (
	streamStateReset uint32 = iota
	streamStateDestroying
	streamStateDestroyed
)

type BaseStream struct {
	sync.Mutex
	streamListeners []types.StreamEventListener

	state uint32
}

func (s *BaseStream) AddEventListener(streamCb types.StreamEventListener) {
	s.Lock()
	s.streamListeners = append(s.streamListeners, streamCb)
	s.Unlock()
}

func (s *BaseStream) RemoveEventListener(streamCb types.StreamEventListener) {
	s.Lock()
	defer s.Unlock()
	cbIdx := -1

	for i, cb := range s.streamListeners {
		if cb == streamCb {
			cbIdx = i
			break
		}
	}

	if cbIdx > -1 {
		s.streamListeners = append(s.streamListeners[:cbIdx], s.streamListeners[cbIdx+1:]...)
	}
}

func (s *BaseStream) ResetStream(reason types.StreamResetReason) {
	if atomic.LoadUint32(&s.state) != streamStateReset {
		return
	}
	defer s.DestroyStream()
	s.Lock()
	defer s.Unlock()

	for _, listener := range s.streamListeners {
		listener.OnResetStream(reason)
	}
}

func (s *BaseStream) DestroyStream() {
	if !atomic.CompareAndSwapUint32(&s.state, streamStateReset, streamStateDestroying) {
		return
	}
	s.Lock()
	defer s.Unlock()
	for _, listener := range s.streamListeners {
		listener.OnDestroyStream()
	}
	atomic.StoreUint32(&s.state, streamStateDestroyed)
}
