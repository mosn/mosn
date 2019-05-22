package stream

import (
	"sync"

	"sofastack.io/sofa-mosn/pkg/types"
)

type BaseStream struct {
	sync.Mutex
	streamListeners []types.StreamEventListener
	// TODO: this is a temporary fix for DestroyStream() concurrency
	once sync.Once
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
	s.once.Do(func() {
		s.Lock()
		defer s.Unlock()
		for _, listener := range s.streamListeners {
			listener.OnDestroyStream()
		}
	})
}
