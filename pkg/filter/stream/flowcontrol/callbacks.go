package flowcontrol

import (
	"context"

	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/mosn/pkg/types"
)

// Callbacks defines the flow control callbacks
type Callbacks interface {
	Init(f *StreamFilter)
	ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) *base.ResourceWrapper
	AfterBlock(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	AfterPass(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	Exit(filter *StreamFilter)
	Enabled() bool
}

// DefaultCallbacks represents the default flow control filter implementation.
type DefaultCallbacks struct{}

// Init is a no-op.
func (nc *DefaultCallbacks) Init(f *StreamFilter) {}

// ParseResource is a no-op.
func (nc *DefaultCallbacks) ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) *base.ResourceWrapper {
	return nil
}

// AfterBlock is a no-op.
func (nc *DefaultCallbacks) AfterBlock(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
}

// AfterPass is a no-op.
func (nc *DefaultCallbacks) AfterPass(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
}

// Exit is a no-op.
func (nc *DefaultCallbacks) Exit(filter *StreamFilter) {}

// Enabled always returns true.
func (nc *DefaultCallbacks) Enabled() bool { return true }
