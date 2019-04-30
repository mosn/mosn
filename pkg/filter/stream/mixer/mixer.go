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

package mixer

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/istio/control/http"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"istio.io/api/mixer/v1/config/client"
)

func init() {
	// static mixer stream filter factory
	filter.RegisterStream(v2.MIXER, CreateMixerFilterFactory)
}

// FilterConfigFactory filter config factory
type FilterConfigFactory struct {
	MixerConfig *v2.Mixer
}

type mixerFilter struct {
	context               context.Context
	config                *v2.Mixer
	serviceContext        *http.ServiceContext
	clientContext         *http.ClientContext
	requestHandler        http.RequestHandler
	receiverFilterHandler types.StreamReceiverFilterHandler
	requestTotalSize      uint64
}

// newMixerFilter used to create new mixer filter
func newMixerFilter(context context.Context, config *v2.Mixer) *mixerFilter {
	filter := &mixerFilter{
		context:       context,
		config:        config,
		clientContext: http.NewClientContext(config),
	}
	filter.serviceContext = http.NewServiceContext(filter.clientContext)
	return filter
}

func (f *mixerFilter) ReadPerRouteConfig(perFilterConfig map[string]interface{}) {
	mixerConfig, exist := perFilterConfig[v2.MIXER]
	if !exist {
		return
	}

	serviceConfig, ok := mixerConfig.(client.ServiceConfig)
	if !ok {
		return
	}

	f.serviceContext.SetServiceConfig(&serviceConfig)
}

func (f *mixerFilter) createRequestHandler() {
	if f.requestHandler != nil {
		log.DefaultLogger.Tracef("requestHandler not nil, return")
		return
	}

	route := f.receiverFilterHandler.Route()
	if route == nil {
		log.DefaultLogger.Tracef("no route, return")
		return
	}
	rule := route.RouteRule()
	if rule == nil {
		log.DefaultLogger.Tracef("no route rule, return")
		return
	}

	perFilterConfig := rule.PerFilterConfig()

	if perFilterConfig != nil {
		f.ReadPerRouteConfig(perFilterConfig)
	}

	f.requestHandler = http.NewRequestHandler(f.serviceContext)
}

func (f *mixerFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) types.StreamFilterStatus {
	if headers != nil {
		f.requestTotalSize += headers.ByteSize()
		f.createRequestHandler()
	}
	if buf != nil {
		f.requestTotalSize += uint64(buf.Len())
	}
	if trailers != nil {
		f.requestTotalSize += trailers.ByteSize()
	}

	return types.StreamFilterContinue
}

func (f *mixerFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *mixerFilter) OnDestroy() {}

func (f *mixerFilter) Log(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	if reqHeaders == nil || respHeaders == nil || requestInfo == nil {
		return
	}

	f.createRequestHandler()

	// TODO: use f.receiverFilterHandler.Connection() to get address instead of requestInfo
	checkData := http.NewCheckData(reqHeaders, requestInfo)

	reportData := http.NewReportData(respHeaders, requestInfo, f.requestTotalSize)

	f.requestHandler.Report(checkData, reportData)
}

// CreateFilterChain for create mixer filter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := newMixerFilter(context, f.MixerConfig)
	callbacks.AddStreamReceiverFilter(filter, types.DownFilterAfterRoute)
	callbacks.AddStreamAccessLog(filter)
}

// CreateMixerFilterFactory for create mixer filter factory
func CreateMixerFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &FilterConfigFactory{
		MixerConfig: config.ParseMixerFilter(conf),
	}, nil
}
