package flowcontrol

import (
	"context"
	"strconv"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"
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
	config *Config
}

// Init is a no-op.
func (dc *DefaultCallbacks) Init() {}

// ParseResource parses resource from context.
func (dc *DefaultCallbacks) ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) *ParsedResource {
	resource, err := variable.GetProtocolResource(ctx, dc.config.KeyType)
	if err != nil || resource == "" {
		log.DefaultLogger.Errorf("parse resource failed: %v", err)
		return nil
	}
	res := base.NewResourceWrapper(resource, base.ResTypeWeb, base.Inbound)
	options := []sentinel.EntryOption{
		sentinel.WithTrafficType(base.Inbound),
	}
	return &ParsedResource{resource: res, opts: options}
}

// AfterBlock sends response directly.
func (dc *DefaultCallbacks) AfterBlock(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
	headers.Set(types.HeaderStatus, strconv.Itoa(dc.config.Action.Status))
	filter.handler.SendDirectResponse(headers, buffer.NewIoBufferString(dc.config.Action.Body), trailers)
}

// AfterPass is a no-op.
func (dc *DefaultCallbacks) AfterPass(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
}

// Exit is a no-op.
func (dc *DefaultCallbacks) Exit(filter *StreamFilter) {}

// Enabled reports whether the callbacks enabled.
func (dc *DefaultCallbacks) Enabled() bool { return dc.config.GlobalSwitch }
