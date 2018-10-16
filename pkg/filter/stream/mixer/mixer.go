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
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	filter.RegisterStream("mixer", CreateMixerFilterFactory)
}

type FilterConfigFactory struct {
	MixerConfig *v2.Mixer
}

type mixerFilter struct {
	context context.Context
	handler http.RequestHandler
	callback types.StreamReceiverFilterCallbacks
	requestTotalSize uint64
}

func NewMixerFilter(context context.Context, config *v2.Mixer) *mixerFilter {
	return &mixerFilter{
		context:       context,
	}
}

func (f *mixerFilter) OnDecodeHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	f.requestTotalSize += headers.ByteSize()
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
	f.callback = cb
}

func (f *mixerFilter) OnDestroy() {}

func (m *mixerFilter) Log(reqHeaders types.HeaderMap, respHeaders types.HeaderMap, requestInfo types.RequestInfo) {
	if m.handler == nil {
		m.handler = http.NewRequestHandler()
	}

	checkData := http.NewCheckData(reqHeaders, requestInfo, m.callback.Connection())

	reportData := http.NewReportData(respHeaders, requestInfo, m.requestTotalSize)

	m.handler.Report(checkData, reportData)
}

func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewMixerFilter(context, f.MixerConfig)
	callbacks.AddStreamReceiverFilter(filter)
	callbacks.AddAccessLog(filter)
}

func CreateMixerFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	factory := &FilterConfigFactory{
		MixerConfig: config.ParseMixerFilter(conf),
	}

	//log.DefaultLogger.Errorf("mix config:%v", factory.MixerConfig.MixerAttributes.Attributes)
	return factory, nil
}
