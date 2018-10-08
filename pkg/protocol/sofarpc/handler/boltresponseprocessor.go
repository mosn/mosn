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

package handler

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type BoltResponseProcessor struct{}
type BoltResponseProcessorV2 struct{}

func (b *BoltResponseProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltResponseCommand); ok {
		deserializeResponse(context, cmd)
		reqID := protocol.StreamIDConv(cmd.ReqID)

		//print tracer log
		log.DefaultLogger.Infof("streamID=%s,protocol=%s", reqID, "bolt")

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				// 回调到stream中的OnDecoderHeader，回传HEADER数据
				//if cmd.Content == nil {
				//	cmd.ResponseHeader[types.HeaderStremEnd] = "yes"
				//}

				status := filter.OnDecodeHeader(reqID, cmd, cmd.Content == nil)

				if status == types.Stop {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.Content), true)

				if status == types.Stop {
					return
				}
			}
		}
	}
}

func (b *BoltResponseProcessorV2) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltV2ResponseCommand); ok {
		deserializeResponse(context, &cmd.BoltResponseCommand)
		reqID := protocol.StreamIDConv(cmd.ReqID)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				//if cmd.Content == nil {
				//	cmd.ResponseHeader[types.HeaderStremEnd] = "yes"
				//}

				status := filter.OnDecodeHeader(reqID, cmd, cmd.Content == nil)

				if status == types.Stop {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.Content), true)

				if status == types.Stop {
					return
				}
			}
		}
	}
}

func deserializeResponse(context context.Context, responseCommand *sofarpc.BoltResponseCommand) {
	//get instance
	serializeIns := serialize.Instance

	//logger
	logger := log.ByContext(context)

	protocolCtx := protocol.ProtocolBuffersByContext(context)
	responseCommand.ResponseHeader = protocolCtx.GetRspHeaders()

	//serialize header
	serializeIns.DeSerialize(responseCommand.HeaderMap, &responseCommand.ResponseHeader)
	logger.Debugf("Deserialize response header map: %+v", responseCommand.ResponseHeader)

	//serialize class name
	serializeIns.DeSerialize(responseCommand.ClassName, &responseCommand.ResponseClass)
	logger.Debugf("Response ClassName is: %s", responseCommand.ResponseClass)
}
