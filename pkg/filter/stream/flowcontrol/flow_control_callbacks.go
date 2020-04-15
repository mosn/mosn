package flowcontrol

import (
	"context"

	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/mosn/pkg/types"
)

// FlowControlCallbacks defines the flow control filter interface
type FlowControlCallbacks interface {
	Init(f *FlowControlStreamFilter)
	ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) *base.ResourceWrapper
	AfterBlock(flowControlFilter *FlowControlStreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	AfterPass(flowControlFilter *FlowControlStreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	Exit(filter *FlowControlStreamFilter)
	Enabled() bool
}

// DefaultCallbacks represents the default flow control filter implementation.
type DefaultCallbacks struct {}

// Init is a no-op.
func (nc *DefaultCallbacks) Init(f *FlowControlStreamFilter) {}

// ParseResource is a no-op.
func (nc *DefaultCallbacks) ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) *base.ResourceWrapper {
	return nil
}

// AfterBlock is a no-op.
func (nc *DefaultCallbacks) AfterBlock(flowControlFilter *FlowControlStreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {}

// AfterPass is a no-op.
func (nc *DefaultCallbacks) AfterPass(flowControlFilter *FlowControlStreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {}

// Exit is a no-op.
func (nc *DefaultCallbacks) Exit(filter *FlowControlStreamFilter) {}

// Enabled always returns true.
func (nc *DefaultCallbacks) Enabled() bool { return true }
