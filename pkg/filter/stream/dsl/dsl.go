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

package dsl

import (
	"context"
	"runtime/debug"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/cel/extract"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

type DSLFilter struct {
	context               context.Context
	receiverFilterHandler api.StreamReceiverFilterHandler
	senderHandler         api.StreamSenderFilterHandler

	dsl *DSL
}

// NewDSLFilter used to create new dsl filter
func NewDSLFilter(ctx context.Context, dsl *DSL) *DSLFilter {
	filter := &DSLFilter{
		context: ctx,
		dsl:     dsl,
	}
	return filter
}

func (f *DSLFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	defer func() {
		if r := recover(); r != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [dsl] OnReceive() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	parentBag := extract.ExtractAttributes(headers, nil, f.receiverFilterHandler.RequestInfo(), buf, trailers, time.Now())
	bag := attribute.NewMutableBag(parentBag)
	bag.Set(extract.KContext, ctx)
	// TODO use f.receiverFilterHandler.GetFilterCurrentPhase
	switch getCurrentPhase(ctx) {
	case api.BeforeRoute:
		if f.dsl.BeforeRouterDSL != nil {
			f.dsl.BeforeRouterDSL.Evaluate(bag)
		}
	case api.AfterRoute:
		if f.dsl.AfterRouterDSL != nil {
			f.dsl.AfterRouterDSL.Evaluate(bag)
		}
	case api.AfterChooseHost:
		if f.dsl.AfterBalancerDSL != nil {
			f.dsl.AfterBalancerDSL.Evaluate(bag)
		}
	}

	return api.StreamFilterContinue
}

func (f *DSLFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	defer func() {
		if r := recover(); r != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [dsl] Append() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	if f.dsl.SendFilterDSL == nil {
		return api.StreamFilterContinue
	}

	parentBag := extract.ExtractAttributes(nil, headers, f.receiverFilterHandler.RequestInfo(), buf, trailers, time.Now())
	bag := attribute.NewMutableBag(parentBag)
	bag.Set(extract.KContext, ctx)
	f.dsl.SendFilterDSL.Evaluate(bag)

	return api.StreamFilterContinue
}

func (f *DSLFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *DSLFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.senderHandler = handler
}

func (f *DSLFilter) OnDestroy() {}

func (f *DSLFilter) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	defer func() {
		if r := recover(); r != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [dsl] Log() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	if f.dsl.LogDSL == nil {
		return
	}

	parentBag := extract.ExtractAttributes(reqHeaders, respHeaders, f.receiverFilterHandler.RequestInfo(), nil, nil, time.Now())
	bag := attribute.NewMutableBag(parentBag)
	bag.Set(extract.KContext, ctx)
	f.dsl.LogDSL.Evaluate(bag)
}

func getCurrentPhase(ctx context.Context) api.FilterPhase {
	// default AfterRoute
	p := api.AfterRoute

	if val := mosnctx.Get(ctx, types.ContextKeyStreamFilterPhase); val != nil {
		if phase, ok := val.(types.Phase); ok {
			switch phase {
			case types.DownFilter:
				p = api.BeforeRoute
			case types.DownFilterAfterRoute:
				p = api.AfterRoute
			case types.DownFilterAfterChooseHost:
				p = api.AfterChooseHost
			}
		}
	}

	return p

}
