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
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type BoltResponseProcessor struct{}
type BoltResponseProcessorV2 struct{}

func (b *BoltResponseProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltResponseCommand); ok {
		deserializeResponseAllFields(context, cmd)
		reqID := protocol.StreamIDConv(cmd.ReqID)

		//print tracer log
		log.DefaultLogger.Infof("streamID=%s,protocol=%s", reqID, "bolt")

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			var content types.IoBuffer
			if cmd.Content != nil {
				protocolCtx := protocol.ProtocolBuffersByContent(context)
				content = protocolCtx.GetRspData(len(cmd.Content))
				content.Write(cmd.Content)
			}

			if cmd.ResponseHeader != nil {
				// 回调到stream中的OnDecoderHeader，回传HEADER数据
				if cmd.Content == nil {
					cmd.ResponseHeader[types.HeaderStremEnd] = "yes"
				}

				status := filter.OnDecodeHeader(reqID, cmd.ResponseHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(reqID, content)

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

func (b *BoltResponseProcessorV2) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltV2ResponseCommand); ok {
		deserializeResponseAllFieldsV2(context, cmd)
		reqID := protocol.StreamIDConv(cmd.ReqID)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				if cmd.Content == nil {
					cmd.ResponseHeader[types.HeaderStremEnd] = "yes"
				}

				status := filter.OnDecodeHeader(reqID, cmd.ResponseHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}
func deserializeResponseAllFields(context context.Context, responseCommand *sofarpc.BoltResponseCommand) {
	//get instance
	serializeIns := serialize.Instance

	//logger
	logger := log.ByContext(context)

	protocolCtx := protocol.ProtocolBuffersByContent(context)
	allField := protocolCtx.GetRspHeaders()

	//serialize header
	//headerMap := sofarpc.GetMap(context, defaultTmpBufferSize)
	serializeIns.DeSerialize(responseCommand.HeaderMap, &allField)
	logger.Debugf("deserialize header map: %+v", allField)

	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(responseCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(responseCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(responseCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(responseCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(responseCommand.ReqID), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(responseCommand.CodecPro), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(responseCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(responseCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(responseCommand.ContentLen), 10)
	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)] = strconv.FormatUint(uint64(responseCommand.ResponseStatus), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespTimeMills)] = strconv.FormatUint(uint64(responseCommand.ResponseTimeMillis), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(responseCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = className
	logger.Debugf("Response ClassName is: %s", className)

	responseCommand.ResponseHeader = allField
}

func deserializeResponseAllFieldsV2(context context.Context, responseCommandV2 *sofarpc.BoltV2ResponseCommand) {
	deserializeResponseAllFields(context, &responseCommandV2.BoltResponseCommand)
	responseCommandV2.ResponseHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(responseCommandV2.Version1), 10)
	responseCommandV2.ResponseHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(responseCommandV2.SwitchCode), 10)
}
