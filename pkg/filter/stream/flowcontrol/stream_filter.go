package flowcontrol

import (
	"context"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// FlowControlFilterName is the flow control stream filter name.
const FlowControlFilterName = "flowControlFilter"

// StreamFilter represents the flow control stream filter.
type StreamFilter struct {
	Entry      *base.SentinelEntry
	BlockError *base.BlockError
	Callbacks  Callbacks
	handler    api.StreamReceiverFilterHandler
}

// NewStreamFilter creates flow control filter.
func NewStreamFilter(callbacks Callbacks) *StreamFilter {
	callbacks.Init()
	return &StreamFilter{Callbacks: callbacks}
}

func (f *StreamFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

// OnReceive creates resource and judges whether current request should be blocked.
func (f *StreamFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if !f.Callbacks.Enabled() {
		return api.StreamFilterContinue
	}
	pr := f.Callbacks.ParseResource(ctx, headers, buf, trailers)
	if pr == nil {
		log.DefaultLogger.Warnf("can't get resource: %+v", headers)
		return api.StreamFilterContinue
	}
	entry, err := sentinel.Entry(pr.resource.Name(), pr.opts...)
	f.Entry = entry
	f.BlockError = err
	if err != nil {
		f.Callbacks.AfterBlock(f, ctx, headers, buf, trailers)
		return api.StreamFilterStop
	}

	f.Callbacks.AfterPass(f, ctx, headers, buf, trailers)
	return api.StreamFilterContinue
}

// OnDestroy does some exit tasks.
func (f *StreamFilter) OnDestroy() {
	if f.Entry != nil {
		f.Callbacks.Exit(f)
		f.Entry.Exit()
	}
}
