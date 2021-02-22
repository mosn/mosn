// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"sync"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/misc"
)

var entryOptsPool = sync.Pool{
	New: func() interface{} {
		return &EntryOptions{
			resourceType: base.ResTypeCommon,
			entryType:    base.Outbound,
			batchCount:   1,
			flag:         0,
			slotChain:    nil,
			args:         nil,
			attachments:  nil,
		}
	},
}

// EntryOptions represents the options of a Sentinel resource entry.
type EntryOptions struct {
	resourceType base.ResourceType
	entryType    base.TrafficType
	batchCount   uint32
	flag         int32
	slotChain    *base.SlotChain
	args         []interface{}
	attachments  map[interface{}]interface{}
}

func (o *EntryOptions) Reset() {
	o.resourceType = base.ResTypeCommon
	o.entryType = base.Outbound
	o.batchCount = 1
	o.flag = 0
	o.slotChain = nil
	o.args = nil
	o.attachments = nil
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

// DEPRECATED: use WithBatchCount instead.
// WithAcquireCount sets the resource entry with the given batch count (by default 1).
func WithAcquireCount(acquireCount uint32) EntryOption {
	return func(opts *EntryOptions) {
		opts.batchCount = acquireCount
	}
}

// WithBatchCount sets the resource entry with the given batch count (by default 1).
func WithBatchCount(batchCount uint32) EntryOption {
	return func(opts *EntryOptions) {
		opts.batchCount = batchCount
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

// WithSlotChain sets the slot chain.
func WithSlotChain(chain *base.SlotChain) EntryOption {
	return func(opts *EntryOptions) {
		opts.slotChain = chain
	}
}

// WithAttachment set the resource entry with the given k-v pair
func WithAttachment(key interface{}, value interface{}) EntryOption {
	return func(opts *EntryOptions) {
		if opts.attachments == nil {
			opts.attachments = make(map[interface{}]interface{}, 8)
		}
		opts.attachments[key] = value
	}
}

// WithAttachment set the resource entry with the given k-v pairs
func WithAttachments(data map[interface{}]interface{}) EntryOption {
	return func(opts *EntryOptions) {
		if opts.attachments == nil {
			opts.attachments = make(map[interface{}]interface{}, len(data))
		}
		for key, value := range data {
			opts.attachments[key] = value
		}
	}
}

// Entry is the basic API of Sentinel.
func Entry(resource string, opts ...EntryOption) (*base.SentinelEntry, *base.BlockError) {
	options := entryOptsPool.Get().(*EntryOptions)
	defer func() {
		options.Reset()
		entryOptsPool.Put(options)
	}()

	for _, opt := range opts {
		opt(options)
	}
	if options.slotChain == nil {
		options.slotChain = misc.GetResourceSlotChain(resource)
		if options.slotChain == nil {
			options.slotChain = GlobalSlotChain()
		}
	}
	return entry(resource, options)
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
	ctx.Input.BatchCount = options.batchCount
	ctx.Input.Flag = options.flag
	if len(options.args) != 0 {
		ctx.Input.Args = options.args
	}
	if len(options.attachments) != 0 {
		ctx.Input.Attachments = options.attachments
	}
	e := base.NewSentinelEntry(ctx, rw, sc)
	ctx.SetEntry(e)
	r := sc.Entry(ctx)
	if r == nil {
		// This indicates internal error in some slots, so just pass
		return e, nil
	}
	if r.Status() == base.ResultStatusBlocked {
		// r will be put to Pool in calling Exit()
		// must finish the lifecycle of r.
		blockErr := base.NewBlockErrorFromDeepCopy(r.BlockError())
		e.Exit()
		return nil, blockErr
	}

	return e, nil
}
