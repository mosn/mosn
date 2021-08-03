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

package grpc

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
)

type grpcStreamFilterChain struct {
	*streamfilter.DefaultStreamFilterChainImpl
	phase types.Phase
	err   error
}

func (sfc *grpcStreamFilterChain) AddStreamSenderFilter(filter api.StreamSenderFilter, phase api.SenderFilterPhase) {
	handler := newStreamSenderFilterHandler(sfc)
	filter.SetSenderFilterHandler(handler)
	sfc.DefaultStreamFilterChainImpl.AddStreamSenderFilter(filter, phase)
}

func (sfc *grpcStreamFilterChain) AddStreamReceiverFilter(filter api.StreamReceiverFilter, phase api.ReceiverFilterPhase) {
	handler := newStreamReceiverFilterHandler(sfc)
	filter.SetReceiveFilterHandler(handler)
	sfc.DefaultStreamFilterChainImpl.AddStreamReceiverFilter(filter, phase)
}

func (sfc *grpcStreamFilterChain) AddStreamAccessLog(accessLog api.AccessLog) {
	sfc.DefaultStreamFilterChainImpl.AddStreamAccessLog(accessLog)
}

// the stream filter chain are not allowed to be used anymore after calling this func
func (sfc *grpcStreamFilterChain) destroy() {
	// filter destroy
	sfc.DefaultStreamFilterChainImpl.OnDestroy()

	// reset fields
	streamfilter.PutStreamFilterChain(sfc.DefaultStreamFilterChainImpl)
	sfc.DefaultStreamFilterChainImpl = nil
}

func (sfc *grpcStreamFilterChain) receiverFilterStatusHandler(phase api.ReceiverFilterPhase, status api.StreamFilterStatus) {

}

func (sfc *grpcStreamFilterChain) senderFilterStatusHandler(phase api.SenderFilterPhase, status api.StreamFilterStatus) {

}
