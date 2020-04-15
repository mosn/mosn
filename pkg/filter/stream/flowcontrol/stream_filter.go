package flowcontrol

import (
	"context"
	"sync"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

var (
	once             sync.Once
	defaultCallbacks = &DefaultCallbacks{}
)

func init() {
	err := sentinel.InitDefault()
	if err != nil {
		log.DefaultLogger.Errorf("init sentinel failed")
		panic(err)
	}
}

// FlowControlStreamFilter represents the flow control stream filter.
type FlowControlStreamFilter struct {
	Entry      *base.SentinelEntry
	BlockError *base.BlockError
	Callbacks  FlowControlCallbacks
}

func (f *FlowControlStreamFilter) init() {
	if f.Callbacks == nil {
		f.Callbacks = defaultCallbacks
	}
	f.Callbacks.Init(f)
}

// OnReceive creates resource and judges whether current request should be blocked.
func (f *FlowControlStreamFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if !f.Callbacks.Enabled() {
		return api.StreamFilterContinue
	}
	resource := f.Callbacks.ParseResource(ctx, headers, buf, trailers)
	if resource == nil {
		log.DefaultLogger.Warnf("can't get resource: %+v", headers)
		return api.StreamFilterContinue
	}

	entry, err := sentinel.Entry(resource.Name(),
		sentinel.WithTrafficType(resource.FlowType()))
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
func (f *FlowControlStreamFilter) OnDestroy() {
	if f.Entry != nil {
		f.Callbacks.Exit(f)
		f.Entry.Exit()
	}
}
