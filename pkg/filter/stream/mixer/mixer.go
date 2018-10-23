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
	"encoding/json"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/istio/control/http"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/jsonpb"
	"istio.io/api/mixer/v1/config/client"

	protobuf_types "github.com/gogo/protobuf/types"
)

const (
	mixerFilterName = "mixer"
)

func init() {
	// static mixer stream filter factory
	filter.RegisterStream(mixerFilterName, CreateMixerFilterFactory)
	// dynamic http_filter mixer config factory
	filter.RegisterNamedHttpFilterConfigFactory(mixerFilterName, CreateMixerConfigFactory)
}

type FilterConfigFactory struct {
	MixerConfig *v2.Mixer
}

type mixerFilter struct {
	context          context.Context
	config           *v2.Mixer
	serviceContext	 *http.ServiceContext
	clientContext 	 *http.ClientContext
	handler          http.RequestHandler
	decodeCallback   types.StreamReceiverFilterCallbacks
	requestTotalSize uint64
}

func NewMixerFilter(context context.Context, config *v2.Mixer) *mixerFilter {
	filter := &mixerFilter{
		context:      context,
		config:				config,
		clientContext:http.NewClientContext(config),
	}
	filter.serviceContext = http.NewServiceContext(filter.clientContext)
	return filter
}

func (f *mixerFilter) ReadPerRouteConfig(perFilterConfig map[string]*v2.PerRouterConfig) {
	mixerConfig, exist := perFilterConfig[mixerFilterName]
	if !exist {
		return
	}

	var serviceConfig client.ServiceConfig
	err := util.StructToMessage(mixerConfig.Struct, &serviceConfig)
	if err != nil {
		return
	}

	f.serviceContext.SetServiceConfig(&serviceConfig)
}

func (f *mixerFilter) createRequestHandler() {
	if f.handler != nil {
		log.DefaultLogger.Infof("handler not nil, return")
		return
	}

	route := f.decodeCallback.Route()
	if route == nil {
		log.DefaultLogger.Infof("no route, return")
		return
	}
	rule := route.RouteRule()
	if rule == nil {
		log.DefaultLogger.Infof("no route rule, return")
		return
	}

	perFilterConfig := rule.PerFilterConfig()

	if perFilterConfig != nil {
		f.ReadPerRouteConfig(perFilterConfig)
	}

	f.handler = http.NewRequestHandler(f.serviceContext)
}

func (f *mixerFilter) OnDecodeHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	f.requestTotalSize += headers.ByteSize()

	f.createRequestHandler()

	return types.StreamHeadersFilterContinue
}

func (f *mixerFilter) OnDecodeData(buf types.IoBuffer, endStream bool) types.StreamDataFilterStatus {
	f.requestTotalSize += uint64(buf.Len())

	return types.StreamDataFilterContinue
}

func (f *mixerFilter) OnDecodeTrailers(trailers types.HeaderMap) types.StreamTrailersFilterStatus {
	f.requestTotalSize += trailers.ByteSize()
	return types.StreamTrailersFilterContinue
}

func (f *mixerFilter) SetDecoderFilterCallbacks(cb types.StreamReceiverFilterCallbacks) {
	f.decodeCallback = cb
}

func (f *mixerFilter) OnDestroy() {}

func (m *mixerFilter) Log(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	//log.DefaultLogger.Infof("in mixer log, config: %v", m.config)

	m.createRequestHandler()

	checkData := http.NewCheckData(reqHeaders, requestInfo, m.decodeCallback.Connection())

	reportData := http.NewReportData(respHeaders, requestInfo, m.requestTotalSize)

	m.handler.Report(checkData, reportData)
}

func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewMixerFilter(context, f.MixerConfig)
	callbacks.AddStreamReceiverFilter(filter)
	callbacks.AddAccessLog(filter)
}

func CreateMixerFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &FilterConfigFactory{
		MixerConfig: config.ParseMixerFilter(conf),
	}, nil
}

// MixerConfigFactory handle dynamic http filter mixer config
type MixerConfigFactory struct {
	Config map[string]interface{}
	MixerConfig v2.Mixer
}

func (m *MixerConfigFactory) CreateFilter() v2.Filter {
	return v2.Filter{
		Type:mixerFilterName,
		Config:m.Config,
	}
}

func CreateMixerConfigFactory(config *protobuf_types.Struct) (types.NamedHttpFilterConfigFactory, error) {
	factory := &MixerConfigFactory {
	}

	err := util.StructToMessage(config, &factory.MixerConfig.HttpClientConfig)
	if err != nil {
		return nil, err
	}

	marshaler := jsonpb.Marshaler{}
	str, err := marshaler.MarshalToString(&factory.MixerConfig.HttpClientConfig)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(str), &factory.Config)

	return factory, err
}