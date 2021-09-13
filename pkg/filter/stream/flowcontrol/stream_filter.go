/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
const (
	FlowControlFilterName = "flowControlFilter"
)

// StreamFilter represents the flow control stream filter.
type StreamFilter struct {
	Entry           *base.SentinelEntry
	BlockError      *base.BlockError
	Callbacks       Callbacks
	ReceiverHandler api.StreamReceiverFilterHandler
	SenderHandler   api.StreamSenderFilterHandler
	trafficType     base.TrafficType
	ranComplete     bool
}

// NewStreamFilter creates flow control filter.
func NewStreamFilter(callbacks Callbacks, trafficType base.TrafficType) *StreamFilter {
	callbacks.Init()
	return &StreamFilter{Callbacks: callbacks, trafficType: trafficType}
}

func (f *StreamFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.ReceiverHandler = handler
}

func (f *StreamFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.SenderHandler = handler
}

// OnReceive creates resource and judges whether current request should be blocked.
func (f *StreamFilter) OnReceive(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if !f.Callbacks.Enabled() || f.Callbacks.ShouldIgnore(f, ctx, headers, buf, trailers) {
		return api.StreamFilterContinue
	}
	f.Callbacks.Prepare(ctx, headers, buf, trailers, f.trafficType)
	pr := f.Callbacks.ParseResource(ctx, headers, buf, trailers, f.trafficType)
	if pr == nil {
		log.DefaultLogger.Warnf("can't get resource: %+v", headers)
		return api.StreamFilterContinue
	}

	entry, err := sentinel.Entry(pr.Resource.Name(), pr.Opts...)
	f.Entry = entry
	f.BlockError = err
	if err != nil {
		f.Callbacks.AfterBlock(f, ctx, headers, buf, trailers)
		return api.StreamFilterStop
	}

	f.Callbacks.AfterPass(f, ctx, headers, buf, trailers)
	return api.StreamFilterContinue
}

// Append will be called after request response, do nothing if request was
// blocked in OnReceive phase.
func (f *StreamFilter) Append(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if f.BlockError != nil {
		return api.StreamFilterStop
	}

	if f.Entry == nil {
		return api.StreamFilterContinue
	}
	f.Callbacks.Append(ctx, headers, buf, trailers)
	return api.StreamFilterContinue
}

// OnDestroy calls the exit tasks.
func (f *StreamFilter) OnDestroy() {
	if f.Entry != nil && !f.ranComplete {
		f.ranComplete = true
		f.Callbacks.Exit(f)
		f.Entry.Exit()
	}
}
