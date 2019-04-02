package stream

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"sync"
)

type BaseStream struct {
	sync.Mutex
	streamListeners []types.StreamEventListener
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
	defer s.DestroyStream()
	s.Lock()
	defer s.Unlock()

	for _, listener := range s.streamListeners {
		listener.OnResetStream(reason)
	}
}

func (s *BaseStream) DestroyStream() {
	s.Lock()
	defer s.Unlock()
	for _, listener := range s.streamListeners {
		listener.OnDestroyStream()
	}
}
