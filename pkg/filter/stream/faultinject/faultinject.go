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
//Similar to Network's damage on flow

package faultinject

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	filter.RegisterStream("fault_inject", CreateFaultInjectFilterFactory)
}

type faultInjectFilter struct {
	context context.Context

	delayPercent  uint32
	delayDuration uint64
	delaying      uint32
	cb            types.StreamReceiverFilterCallbacks
}

func NewFaultInjectFilter(context context.Context, config *v2.FaultInject) types.StreamReceiverFilter {
	return &faultInjectFilter{
		context:       context,
		delayPercent:  config.DelayPercent,
		delayDuration: config.DelayDuration,
	}
}

func (f *faultInjectFilter) OnDecodeHeaders(headers map[string]string, endStream bool) types.FilterHeadersStatus {
	f.tryInjectDelay()

	if atomic.LoadUint32(&f.delaying) > 0 {
		return types.FilterHeadersStatusStopIteration
	}

	return types.FilterHeadersStatusContinue
}

func (f *faultInjectFilter) OnDecodeData(buf types.IoBuffer, endStream bool) types.FilterDataStatus {
	f.tryInjectDelay()

	if atomic.LoadUint32(&f.delaying) > 0 {
		return types.FilterDataStatusStopIterationAndBuffer
	}

	return types.FilterDataStatusContinue
}

func (f *faultInjectFilter) OnDecodeTrailers(trailers map[string]string) types.FilterTrailersStatus {
	f.tryInjectDelay()

	if atomic.LoadUint32(&f.delaying) > 0 {
		return types.FilterTrailersStatusStopIteration
	}

	return types.FilterTrailersStatusContinue
}

func (f *faultInjectFilter) SetDecoderFilterCallbacks(cb types.StreamReceiverFilterCallbacks) {
	f.cb = cb
}

func (f *faultInjectFilter) OnDestroy() {}

func (f *faultInjectFilter) tryInjectDelay() {
	if atomic.LoadUint32(&f.delaying) > 0 {
		return
	}

	duration := f.getDelayDuration()

	if duration > 0 {
		if atomic.CompareAndSwapUint32(&f.delaying, 0, 1) {
			go func() {
				select {
				case <-time.After(time.Duration(duration) * time.Millisecond):
					atomic.StoreUint32(&f.delaying, 0)
					log.ByContext(f.context).Debugf("[FaultInject] Continue after delay")
					f.cb.ContinueDecoding()
				}
			}()
		}
	}
}

func (f *faultInjectFilter) getDelayDuration() uint64 {
	if f.delayPercent == 0 {
		return 0
	}

	if uint32(rand.Intn(100))+1 > f.delayPercent {
		return 0
	}

	return f.delayDuration
}

// ~~ factory
type FilterConfigFactory struct {
	FaultInject *v2.FaultInject
}

func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks types.StreamFilterChainFactoryCallbacks) {
	filter := NewFaultInjectFilter(context, f.FaultInject)
	callbacks.AddStreamReceiverFilter(filter)
}

func CreateFaultInjectFilterFactory(conf map[string]interface{}) (types.StreamFilterChainFactory, error) {
	return &FilterConfigFactory{
		FaultInject: config.ParseFaultInjectFilter(conf),
	}, nil
}
