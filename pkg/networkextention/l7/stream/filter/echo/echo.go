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

package echo

import (
	"context"
	"runtime/debug"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
)

type EchoFilter struct {
	echoer                *FilterConfigFactory
	receiverFilterHandler api.StreamReceiverFilterHandler
	senderHandler         api.StreamSenderFilterHandler
}

// NewEchoFilter used to create new echo filter
func NewEchoFilter(f *FilterConfigFactory) *EchoFilter {
	filter := &EchoFilter{
		echoer: f,
	}
	return filter
}

func (f *EchoFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	defer func() {
		if r := recover(); r != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [echo] OnReceive() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	if f.echoer == nil || f.echoer.Status == 0 {
		return api.StreamFilterContinue
	}

	if f.echoer.Body != "" {
		f.receiverFilterHandler.SendHijackReplyWithBody(f.echoer.Status, protocol.CommonHeader(f.echoer.Headers), f.echoer.Body)
	} else {
		f.receiverFilterHandler.SendHijackReply(f.echoer.Status, protocol.CommonHeader(f.echoer.Headers))
	}

	return api.StreamFilterStop
}

func (f *EchoFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {

	return api.StreamFilterContinue
}

func (f *EchoFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *EchoFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.senderHandler = handler
}

func (f *EchoFilter) OnDestroy() {}

func (f *EchoFilter) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {

}
