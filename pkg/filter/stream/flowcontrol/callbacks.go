package flowcontrol

import (
	"context"
	"strconv"
	"sync"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

var callbacksRegistry = new(sync.Map)

// Callbacks defines the flow control callbacks
type Callbacks interface {
	Init()
	ShouldIgnore(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) bool
	Prepare(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap, trafficType base.TrafficType)
	ParseResource(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap, trafficType base.TrafficType) *ParsedResource
	AfterBlock(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	AfterPass(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	Append(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap)
	Exit(filter *StreamFilter)
	Enabled() bool
	SetConfig(conf *Config)
	GetConfig() *Config
}

// ParsedResource contains the parsed Resource wrapper and entry options.
type ParsedResource struct {
	Resource *base.ResourceWrapper
	Opts     []sentinel.EntryOption
}

// DefaultCallbacks represents the default flow control filter implementation.
type DefaultCallbacks struct {
	config *Config
}

// RegisterCallbacks stores customized callbacks.
func RegisterCallbacks(name string, cb Callbacks) {
	callbacksRegistry.Store(name, cb)
}

// GetCallbacksByName returns specified or default Callbacks.
func GetCallbacksByConfig(conf *Config) Callbacks {
	cb, ok := callbacksRegistry.Load(conf.CallbackName)
	if !ok || cb == nil {
		return &DefaultCallbacks{config: conf}
	}
	customizedCallbacks := cb.(Callbacks)
	customizedCallbacks.SetConfig(conf)
	return customizedCallbacks
}

// Init is a no-op.
func (dc *DefaultCallbacks) Init() {}

// ParseResource parses Resource from context.
func (dc *DefaultCallbacks) ParseResource(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap, trafficType base.TrafficType) *ParsedResource {
	resource, err := variable.GetProtocolResource(ctx, dc.config.KeyType)
	if err != nil || resource == "" {
		log.DefaultLogger.Errorf("parse resource failed: %v", err)
		return nil
	}
	res := base.NewResourceWrapper(resource, base.ResTypeWeb, base.Inbound)
	options := []sentinel.EntryOption{
		sentinel.WithTrafficType(base.Inbound),
	}
	return &ParsedResource{Resource: res, Opts: options}
}

// Prepare is a no-op by default.
func (dc *DefaultCallbacks) Prepare(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap, trafficType base.TrafficType) {
}

// AfterBlock sends response directly.
func (dc *DefaultCallbacks) AfterBlock(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
	variable.SetVariableValue(ctx, types.VarHeaderStatus, strconv.Itoa(dc.config.Action.Status))
	filter.ReceiverHandler.SendDirectResponse(headers, buffer.NewIoBufferString(dc.config.Action.Body), trailers)
}

// AfterPass is a no-op by default.
func (dc *DefaultCallbacks) AfterPass(filter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) {
}

// Append is a no-op by default.
func (dc *DefaultCallbacks) Append(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap) {
}

// Exit is a no-op by default.
func (dc *DefaultCallbacks) Exit(filter *StreamFilter) {}

// Enabled reports whether the callbacks enabled.
func (dc *DefaultCallbacks) Enabled() bool { return dc.config.GlobalSwitch }

// ShouldIgnore is a no-op by default.
func (dc *DefaultCallbacks) ShouldIgnore(flowControlFilter *StreamFilter, ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) bool {
	return false
}

// SetConfig sets the config of callbacks.
func (dc *DefaultCallbacks) SetConfig(conf *Config) {
	dc.config = conf
}

// GetConfig gets the config of callbacks.
func (dc *DefaultCallbacks) GetConfig() *Config {
	return dc.config
}
