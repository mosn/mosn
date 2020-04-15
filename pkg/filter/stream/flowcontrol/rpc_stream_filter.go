package flowcontrol

import (
	"mosn.io/api"
)

// DefaultFlowControlStreamFilter represents the default flow control stream filter.
type DefaultFlowControlStreamFilter struct {
	FlowControlStreamFilter
	handler api.StreamReceiverFilterHandler
}

// NewDefaultFlowControlStreamFilter creates RPC flow control filter.
func NewDefaultFlowControlStreamFilter() *DefaultFlowControlStreamFilter {
	filter := &DefaultFlowControlStreamFilter{FlowControlStreamFilter: FlowControlStreamFilter{}}
	filter.init()
	return filter
}

func (rc *DefaultFlowControlStreamFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	rc.handler = handler
}
