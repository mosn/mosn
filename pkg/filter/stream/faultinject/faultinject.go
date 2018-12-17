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

package faultinject

import (
	"context"
	"math/rand"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/types"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// faultInjectConfig is parsed from v2.StreamFaultInject
type faultInjectConfig struct {
	fixedDelay   time.Duration
	delayPercent uint32
	abortStatus  int
	abortPercent uint32
	upstream     string
	headers      []*types.HeaderData
}

func makefaultInjectConfig(cfg *v2.StreamFaultInject) *faultInjectConfig {
	faultConfig := &faultInjectConfig{
		upstream: cfg.UpstreamCluster,
		headers:  router.GetRouterHeaders(cfg.Headers),
	}
	if cfg.Delay != nil {
		faultConfig.fixedDelay = cfg.Delay.Delay
		faultConfig.delayPercent = cfg.Delay.Percent
	}
	if cfg.Abort != nil {
		faultConfig.abortStatus = cfg.Abort.Status
		faultConfig.abortPercent = cfg.Abort.Percent
	}
	return faultConfig
}

// TODO: this is a hack for per route config parse
// delete it later, when per route config changes to map[string]interface{}
func parseStreamFaultInjectConfig(c interface{}) (*faultInjectConfig, bool) {
	conf := make(map[string]interface{})
	b, err := json.Marshal(c)
	if err != nil {
		log.DefaultLogger.Errorf("config is not a json, %v", err)
		return nil, false
	}
	json.Unmarshal(b, &conf)
	cfg, err := config.ParseStreamFaultInjectFilter(conf)
	if err != nil {
		log.DefaultLogger.Errorf("config is not stream fault inject", err)
		return nil, false
	}
	return makefaultInjectConfig(cfg), true
}

// streamFaultInjectFilter is an implement of types.StreamReceiverFilter
type streamFaultInjectFilter struct {
	ctx       context.Context
	isDelayed bool
	handler   types.StreamReceiverFilterHandler
	config    *faultInjectConfig
	stop      chan struct{}
	rander    *rand.Rand
	headers   types.HeaderMap
}

func NewFilter(ctx context.Context, cfg *v2.StreamFaultInject) types.StreamReceiverFilter {
	log.DefaultLogger.Debugf("create a new fault inject filter")
	return &streamFaultInjectFilter{
		ctx:       ctx,
		isDelayed: false,
		config:    makefaultInjectConfig(cfg),
		stop:      make(chan struct{}),
		rander:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ReadPerRouteConfig makes route-level configuration override filter-level configuration
func (f *streamFaultInjectFilter) ReadPerRouteConfig(cfg map[string]interface{}) {
	if cfg == nil {
		return
	}
	if fault, ok := cfg[v2.FaultStream]; ok {
		if config, ok := parseStreamFaultInjectConfig(fault); ok {
			log.DefaultLogger.Debugf("use router config to replace stream filter config, config: %v", fault)
			f.config = config
		}
	}
}

func (f *streamFaultInjectFilter) SetReceiveFilterHandler(handler types.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *streamFaultInjectFilter) OnReceiveHeaders(headers types.HeaderMap, endStream bool) types.StreamHeadersFilterStatus {
	log.DefaultLogger.Debugf("fault inject filter do receive headers")
	if route := f.handler.Route(); route != nil {
		// TODO: makes ReadPerRouteConfig as the StreamReceiverFilter's function
		f.ReadPerRouteConfig(route.RouteRule().PerFilterConfig())
	}
	if !f.matchUpstream() {
		log.DefaultLogger.Debugf("upstream is not matched")
		return types.StreamHeadersFilterContinue
	}
	// TODO: check downstream nodes, support later
	//if !f.downstreamNodes() {
	//	return types.StreamHeadersFilterContinue
	//}
	if !router.ConfigUtilityInst.MatchHeaders(headers, f.config.headers) {
		log.DefaultLogger.Debugf("header is not matched, request headers: %v, config headers: %v", headers, f.config.headers)
		return types.StreamHeadersFilterContinue
	}
	// TODO: some parameters can get from request header
	if delay := f.getDelayDuration(); delay > 0 {
		f.isDelayed = true
		go func() { // start a timer
			log.DefaultLogger.Debugf("start a delay timer")
			select {
			case <-time.After(delay):
				f.doDelayInject(headers)
			case <-f.stop:
				log.DefaultLogger.Debugf("timer is stopped")
				return
			}
		}()
		// TODO: stats
		f.handler.RequestInfo().SetResponseFlag(types.DelayInjected)
		return types.StreamHeadersFilterStop
	}
	if f.isAbort() {
		f.abort(headers)
		return types.StreamHeadersFilterStop
	}
	return types.StreamHeadersFilterContinue
}

func (f *streamFaultInjectFilter) OnReceiveData(buf types.IoBuffer, endStream bool) types.StreamDataFilterStatus {
	if !f.isDelayed {
		return types.StreamDataFilterContinue
	}
	return types.StreamDataFilterStopAndBuffer
}

func (f *streamFaultInjectFilter) OnReceiveTrailers(trailers types.HeaderMap) types.StreamTrailersFilterStatus {
	if !f.isDelayed {
		return types.StreamTrailersFilterContinue
	}
	return types.StreamTrailersFilterStop
}

func (f *streamFaultInjectFilter) OnDestroy() {
	close(f.stop)
}

// matches and inject

func (f *streamFaultInjectFilter) matchUpstream() bool {
	if f.config.upstream != "" {
		if route := f.handler.Route(); route != nil {
			log.DefaultLogger.Debugf("current cluster name %s, fault inject cluster name %s", route.RouteRule().ClusterName(), f.config.upstream)
			return route.RouteRule().ClusterName() == f.config.upstream
		}
	}
	log.DefaultLogger.Debugf("no upstream in config, returns true")
	return true
}

func (f *streamFaultInjectFilter) getDelayDuration() time.Duration {
	// percent is 0 or delay is 0 means no delay
	if f.config.delayPercent == 0 || f.config.fixedDelay == 0 {
		log.DefaultLogger.Debugf("no delay inject")
		return 0
	}
	// rander generates 0~99, if greater than percent means no delay
	if (f.rander.Uint32() % 100) >= f.config.delayPercent {
		log.DefaultLogger.Debugf("delay percent is not matched")
		return 0
	}
	return f.config.fixedDelay
}

func (f *streamFaultInjectFilter) doDelayInject(headers types.HeaderMap) {
	// abort maybe called after delay
	if f.isAbort() {
		f.abort(headers)
	} else {
		f.handler.ContinueReceiving()
	}
}

func (f *streamFaultInjectFilter) isAbort() bool {
	// percent is 0 means no abort
	if f.config.abortPercent == 0 {
		log.DefaultLogger.Debugf("no abort inject")
		return false
	}
	if (f.rander.Uint32() % 100) >= f.config.abortPercent {
		log.DefaultLogger.Debugf("abort percent is not matched")
		return false
	}
	return true
}

// TODO: make a header
func (f *streamFaultInjectFilter) abort(headers types.HeaderMap) {
	log.DefaultLogger.Debugf("abort inject")
	f.handler.RequestInfo().SetResponseFlag(types.FaultInjected)
	f.handler.SendHijackReply(f.config.abortStatus, headers)
}
