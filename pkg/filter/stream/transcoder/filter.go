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

package transcoder

import (
	"context"
	"mosn.io/mosn/pkg/filter/stream/transcoder/rules"
	"net/http"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// transcodeFilter is an implement of types.StreamReceiverFilter/types.StreamSendFilter
type transcodeFilter struct {
	ctx context.Context
	cfg *config

	transcoder Transcoder

	needTranscode bool

	receiveHandler api.StreamReceiverFilterHandler
	sendHandler    api.StreamSenderFilterHandler
}

func newTranscodeFilter(ctx context.Context, cfg *config) *transcodeFilter {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "[stream filter][transcoder] create transcoder filter with config: %v", cfg)
	}

	transcoder := GetTranscoder(cfg.Type).(Transcoder)
	if !initTranscodePlugin(ctx, cfg.GoPluginConfig) && transcoder == nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] create failed, no such transcoder type: %s", cfg.Type)
		return nil
	}

	return &transcodeFilter{
		ctx:        ctx,
		cfg:        cfg,
		transcoder: transcoder,
	}
}

func initTranscodePlugin(ctx context.Context, cfg *transcodeGoPluginConfig) bool {
	if cfg == nil {
		return false
	}
	listenerName := mosnctx.Get(ctx, types.ContextKeyListenerName).(string)
	transcoders, result := GetInstanceTransferPluginManger().GetTransferPlugin(listenerName)
	if !result {
		transcoders = cfg.Transcoders
	}
	for _, transcoder := range transcoders {
		transcoder.CreateTranscoder(listenerName)
	}

	return true
}

// ReadPerRouteConfig makes route-level configuration override filter-level configuration
func (f *transcodeFilter) readPerRouteConfig(ctx context.Context, cfg map[string]interface{}) {
	if cfg == nil {
		return
	}
	if transcodeCfg, ok := cfg[v2.Transcoder]; ok {
		if config, err := parseConfig(transcodeCfg); err == nil {
			if log.Proxy.GetLogLevel() >= log.DEBUG {
				log.Proxy.Debugf(ctx, "[stream filter][transcoder] use router config to replace stream filter config, config: %v", config)
			}
			f.cfg = config
		}
	}
}

func (f *transcodeFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiveHandler = handler
}

func (f *transcodeFilter) OnReceive(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {

	var outHeaders, outTrailers types.HeaderMap
	var outBuf types.IoBuffer
	var err error

	if ruleInfo, ok := f.Matches(ctx, headers); ok {
		srcPro := mosnctx.Get(ctx, types.ContextKeyDownStreamProtocol).(api.ProtocolName)
		dstPro := ruleInfo.UpstreamSubProtocol
		listenerName := mosnctx.Get(ctx, types.ContextKeyListenerName).(string)
		pluginName := listenerName + "_" + string(srcPro) + "_" + dstPro

		//select transcoder
		transcoder := GetTranscoder(pluginName)

		if transcoder == nil {
			log.Proxy.Errorf(ctx, "[stream filter][transcoder] cloud not found transcoder")
			return api.StreamFilterContinue
		}

		//set sub protocol
		mosnctx.WithValue(ctx, types.ContextSubProtocol, dstPro)
		//set upstream protocol
		mosnctx.WithValue(ctx, types.ContextKeyUpStreamProtocol, ruleInfo.UpstreamProtocol)

		// do transcoding
		outHeaders, outBuf, outTrailers, err = transcoder.(TranscoderSo).TranscodingRequest(ctx, headers, buf, trailers)

	} else if f.transcoder != nil {
		// check accept
		if !f.transcoder.Accept(ctx, headers, buf, trailers) {
			return api.StreamFilterContinue
		}

		// for response check
		f.needTranscode = true

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx, "[stream filter][transcoder] receive request: %+v", headers)
		}

		if route := f.receiveHandler.Route(); route != nil {
			// TODO: makes ReadPerRouteConfig as the StreamReceiverFilter's function
			f.readPerRouteConfig(ctx, route.RouteRule().PerFilterConfig())
		}
		// do transcoding
		outHeaders, outBuf, outTrailers, err = f.transcoder.TranscodingRequest(ctx, headers, buf, trailers)
	}

	if err != nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] transcoder request failed: %v", err)
		f.receiveHandler.RequestInfo().SetResponseFlag(RequestTranscodeFail)
		f.receiveHandler.SendHijackReply(http.StatusBadRequest, headers)
		return api.StreamFilterStop
	}

	f.receiveHandler.SetRequestHeaders(outHeaders)
	f.receiveHandler.SetRequestData(outBuf)
	f.receiveHandler.SetRequestTrailers(outTrailers)
	f.receiveHandler.SetConvert(false)
	return api.StreamFilterContinue
}

func (f *transcodeFilter) OnDestroy() {}

// SetSenderFilterHandler sets the StreamSenderFilterHandler
func (f *transcodeFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.sendHandler = handler
}

// Append encodes request/response
func (f *transcodeFilter) Append(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {

	var outHeaders, outTrailers types.HeaderMap
	var outBuf types.IoBuffer
	var err error

	if ruleInfo, ok := f.Matches(ctx, headers); ok {
		srcPro := mosnctx.Get(ctx, types.ContextKeyDownStreamProtocol).(api.ProtocolName)
		dsrPro := ruleInfo.UpstreamSubProtocol
		listenerName := mosnctx.Get(ctx, types.ContextKeyListenerName).(string)
		pluginName := listenerName + "_" + string(srcPro) + "_" + dsrPro

		//select transcoder
		transcoder := GetTranscoder(pluginName).(TranscoderSo)

		if transcoder == nil {
			log.Proxy.Errorf(ctx, "[stream filter][transcoder] cloud not found transcoder")
			return api.StreamFilterContinue
		}

		// do transcoding
		outHeaders, outBuf, outTrailers, err = transcoder.TranscodingResponse(ctx, headers, buf, trailers)

	} else if f.transcoder != nil {

		if !f.needTranscode {
			return api.StreamFilterContinue
		}

		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx, "[stream filter][transcoder] receive response: %+v", headers)
		}

		// do transcoding
		outHeaders, outBuf, outTrailers, err = f.transcoder.TranscodingResponse(ctx, headers, buf, trailers)
	}

	if err != nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] transcoder response failed: %v", err)
		f.receiveHandler.RequestInfo().SetResponseFlag(RequestTranscodeFail)
		f.receiveHandler.SendHijackReply(http.StatusInternalServerError, headers)
		return api.StreamFilterStop
	}
	f.sendHandler.SetResponseHeaders(outHeaders)
	f.sendHandler.SetResponseData(outBuf)
	f.sendHandler.SetResponseTrailers(outTrailers)
	return api.StreamFilterContinue
}

func (f *transcodeFilter) Matches(ctx context.Context, headers types.HeaderMap) (*rules.RuleInfo, bool) {

	listenerName := mosnctx.Get(ctx, types.ContextKeyListenerName).(string)
	transferRuleConfigs, result := rules.GetInstanceTransferRuleManger().GetTransferRule(listenerName)
	if !result {
		transferRuleConfigs = f.cfg.Rules
	}
	log.DefaultLogger.Infof("[stream filter][transcoder] result %s, transferRuleConfigs %+v", result, transferRuleConfigs)
	if transferRuleConfigs == nil {
		return nil, false
	}
	for _, t := range transferRuleConfigs {
		rule, match := t.Matches(ctx, headers)
		log.DefaultLogger.Infof("[stream filter][transcoder] match %s, rule %+v", match, rule)
		if match {
			return rule, match
		}
	}

	return nil, false
}
