package stream

import (
	"github.com/alipay/sofa-mosn/pkg/types"
)

type BaseStream struct {
	streamListeners []types.StreamEventListener
}

func (s *BaseStream) AddEventListener(streamCb types.StreamEventListener) {
	s.streamListeners = append(s.streamListeners, streamCb)
}

func (s *BaseStream) RemoveEventListener(streamCb types.StreamEventListener) {
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

	for _, listener := range s.streamListeners {
		listener.OnResetStream(reason)
	}
}

func (s *BaseStream) DestroyStream() {
	for _, listener := range s.streamListeners {
		listener.OnDestroyStream()
	}
}
