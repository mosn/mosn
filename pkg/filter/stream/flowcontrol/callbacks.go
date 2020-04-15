package flowcontrol

import (
	"context"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/mosn/pkg/types"
)

// Callbacks defines the flow control callbacks
type Callbacks interface {
	Init()
	ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) *ParsedResource
	AfterBlock(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	AfterPass(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	Exit(filter *StreamFilter)
	Enabled() bool
}

// ParsedResource contains the parsed resource wrapper and entry options.
type ParsedResource struct {
	resource *base.ResourceWrapper
	opts     []sentinel.EntryOption
}

// DefaultCallbacks represents the default flow control filter implementation.
type DefaultCallbacks struct {
	enabled *bool
}

// Init is a no-op.
func (dc *DefaultCallbacks) Init() {}

// ParseResource is a no-op.
func (dc *DefaultCallbacks) ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) *ParsedResource {
	return nil
}

// AfterBlock is a no-op.
func (dc *DefaultCallbacks) AfterBlock(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
}

// AfterPass is a no-op.
func (dc *DefaultCallbacks) AfterPass(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
}

// Exit is a no-op.
func (dc *DefaultCallbacks) Exit(filter *StreamFilter) {}

// Enabled reports whether the callbacks enabled.
func (dc *DefaultCallbacks) Enabled() bool { return *dc.enabled }
