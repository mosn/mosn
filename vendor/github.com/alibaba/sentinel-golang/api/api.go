package api

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

// EntryOptions represents the options of a Sentinel resource entry.
type EntryOptions struct {
	resourceType base.ResourceType
	entryType    base.TrafficType
	acquireCount uint32
	flag         int32
	slotChain    *base.SlotChain
	args         []interface{}
	attachments  map[interface{}]interface{}
}

type EntryOption func(*EntryOptions)

// WithResourceType sets the resource entry with the given resource type.
func WithResourceType(resourceType base.ResourceType) EntryOption {
	return func(opts *EntryOptions) {
		opts.resourceType = resourceType
	}
}

// WithTrafficType sets the resource entry with the given traffic type.
func WithTrafficType(entryType base.TrafficType) EntryOption {
	return func(opts *EntryOptions) {
		opts.entryType = entryType
	}
}

// WithAcquireCount sets the resource entry with the given batch count (by default 1).
func WithAcquireCount(acquireCount uint32) EntryOption {
	return func(opts *EntryOptions) {
		opts.acquireCount = acquireCount
	}
}

// WithFlag sets the resource entry with the given additional flag.
func WithFlag(flag int32) EntryOption {
	return func(opts *EntryOptions) {
		opts.flag = flag
	}
}

// WithArgs sets the resource entry with the given additional parameters.
func WithArgs(args ...interface{}) EntryOption {
	return func(opts *EntryOptions) {
		opts.args = append(opts.args, args...)
	}
}

// WithAttachment set the resource entry with the given k-v pair
func WithAttachment(key interface{}, value interface{}) EntryOption {
	return func(opts *EntryOptions) {
		opts.attachments[key] = value
	}
}

// WithAttachment set the resource entry with the given k-v pairs
func WithAttachments(data map[interface{}]interface{}) EntryOption {
	return func(opts *EntryOptions) {
		for key, value := range data {
			opts.attachments[key] = value
		}
	}
}

// Entry is the basic API of Sentinel.
func Entry(resource string, opts ...EntryOption) (*base.SentinelEntry, *base.BlockError) {
	var options = EntryOptions{
		resourceType: base.ResTypeCommon,
		entryType:    base.Outbound,
		acquireCount: 1,
		flag:         0,
		slotChain:    globalSlotChain,
		args:         []interface{}{},
		attachments:  make(map[interface{}]interface{}),
	}
	for _, opt := range opts {
		opt(&options)
	}

	return entry(resource, &options)
}

func entry(resource string, options *EntryOptions) (*base.SentinelEntry, *base.BlockError) {
	rw := base.NewResourceWrapper(resource, options.resourceType, options.entryType)
	sc := options.slotChain

	if sc == nil {
		return base.NewSentinelEntry(nil, rw, nil), nil
	}
	// Get context from pool.
	ctx := sc.GetPooledContext()
	ctx.Resource = rw
	ctx.Input = &base.SentinelInput{
		AcquireCount: options.acquireCount,
		Flag:         options.flag,
		Args:         options.args,
		Attachments:  options.attachments,
	}

	e := base.NewSentinelEntry(ctx, rw, sc)

	r := sc.Entry(ctx)
	if r == nil {
		// This indicates internal error in some slots, so just pass
		return e, nil
	}
	if r.Status() == base.ResultStatusBlocked {
		e.Exit()
		return nil, r.BlockError()
	}

	return e, nil
}
