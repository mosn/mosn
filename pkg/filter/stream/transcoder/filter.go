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
	"net/http"

	"mosn.io/api"
	"mosn.io/api/extensions/transcoder"

	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/filter/stream/transcoder/matcher"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// transcodeFilter is an implement of types.StreamReceiverFilter/types.StreamSendFilter
type transcodeFilter struct {
	ctx context.Context
	cfg *config

	transcoder    transcoder.Transcoder
	needTranscode bool

	receiveHandler api.StreamReceiverFilterHandler
	sendHandler    api.StreamSenderFilterHandler
}

func newTranscodeFilter(ctx context.Context, cfg *config) *transcodeFilter {
	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(ctx, "[stream filter][transcoder] create transcoder filter with config: %v", cfg)
	}

	return &transcodeFilter{
		ctx: ctx,
		cfg: cfg,
	}
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

	ruleInfo, ok := matcher.TransCoderMatches(ctx, headers, f.cfg.Rules)
	if !ok {
		return api.StreamFilterContinue
	}
	srcPro := mosnctx.Get(ctx, types.ContextKeyDownStreamProtocol).(api.ProtocolName)
	dstPro := ruleInfo.UpstreamSubProtocol
	//select transcoder
	transcoderFactory := GetTranscoderFactory(ruleInfo.GetType(srcPro))
	if transcoderFactory == nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] cloud not found transcoderFactory")
		return api.StreamFilterContinue
	}

	transcoder := transcoderFactory(ruleInfo.Config)
	if transcoder == nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] create transcoder failed")
		return api.StreamFilterContinue
	}

	// check accept
	if !transcoder.Accept(ctx, headers, buf, trailers) {
		return api.StreamFilterContinue
	}

	// for response check
	f.needTranscode = true
	f.transcoder = transcoder

	//TODO set transcoder config
	//set sub protocol
	mosnctx.WithValue(ctx, types.ContextSubProtocol, dstPro)
	//set upstream protocol
	mosnctx.WithValue(ctx, types.ContextKeyUpStreamProtocol, ruleInfo.UpstreamProtocol)

	outHeaders, outBuf, outTrailers, err := transcoder.TranscodingRequest(ctx, headers, buf, trailers)

	if err != nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] transcoder request failed: %v", err)
		f.receiveHandler.RequestInfo().SetResponseFlag(RequestTranscodeFail)
		f.receiveHandler.SendHijackReply(http.StatusBadRequest, headers)
		return api.StreamFilterStop
	}

	f.receiveHandler.SetRequestHeaders(outHeaders)
	f.receiveHandler.SetRequestData(outBuf)
	f.receiveHandler.SetRequestTrailers(outTrailers)

	return api.StreamFilterContinue
}

func (f *transcodeFilter) OnDestroy() {}

// SetSenderFilterHandler sets the StreamSenderFilterHandler
func (f *transcodeFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.sendHandler = handler
}

// Append encodes request/response
func (f *transcodeFilter) Append(ctx context.Context, headers types.HeaderMap, buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if !f.needTranscode {
		return api.StreamFilterContinue
	}
	//select transcoder
	transcoder := f.transcoder

	if transcoder == nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] cloud not found transcoder")
		return api.StreamFilterContinue
	}

	// do transcoding
	outHeaders, outBuf, outTrailers, err := transcoder.TranscodingResponse(ctx, headers, buf, trailers)

	if err != nil {
		log.Proxy.Errorf(ctx, "[stream filter][transcoder] transcoder response failed: %v", err)
		f.receiveHandler.RequestInfo().SetResponseFlag(RequestTranscodeFail)
		f.receiveHandler.SendHijackReply(http.StatusInternalServerError, headers)
		f.needTranscode = false // send hijack should not transcode now.
		return api.StreamFilterStop
	}
	f.sendHandler.SetResponseHeaders(outHeaders)
	f.sendHandler.SetResponseData(outBuf)
	f.sendHandler.SetResponseTrailers(outTrailers)

	return api.StreamFilterContinue
}
