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
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/models"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var defaultTmpBufferSize = 1 << 6

// BoltRequestProcessor process bolt v1 request msg
type BoltRequestProcessor struct{}

// BoltRequestProcessorV2 process bolt v2 request msg
type BoltRequestProcessorV2 struct{}

// Process boltv1 request cmd to map[string]string header
func (b *BoltRequestProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(context, cmd)
		streamID := protocol.GenerateIDString()
		//print tracer log
		log.DefaultLogger.Debugf("time=%s,tracerID=%s,streamID=%s,protocol=%s,service=%s,callerIp=%s", time.Now(), cmd.RequestHeader[models.TRACER_ID_KEY], streamID, cmd.RequestHeader[models.SERVICE_KEY], "bolt", cmd.RequestHeader[models.CALLER_IP_KEY])


		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			var content types.IoBuffer
			if cmd.Content != nil {
				protocolCtx := protocol.ProtocolBuffersByContent(context)
				content = protocolCtx.GetReqData(len(cmd.Content))
				content.Write(cmd.Content)
			}

			if cmd.RequestHeader != nil {
				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				if cmd.Content == nil {
					cmd.RequestHeader[types.HeaderStremEnd] = "yes"
				}

				status := filter.OnDecodeHeader(streamID, cmd.RequestHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamID, content)

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

// Process boltv2 request cmd to map[string]string header
// ctx = type.serverStreamConnection
func (b *BoltRequestProcessorV2) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltV2RequestCommand); ok {
		deserializeRequestAllFieldsV2(context, cmd)
		streamID := protocol.GenerateIDString()

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {

				if cmd.Content == nil {
					cmd.RequestHeader["x-mosn-endstream"] = "yes"
				}

				status := filter.OnDecodeHeader(streamID, cmd.RequestHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamID, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

//Convert BoltV1's Protocol Header  and Content Header to Map[string]string
func deserializeRequestAllFields(context context.Context, requestCommand *sofarpc.BoltRequestCommand) {
	//get instance
	serializeIns := serialize.Instance

	protocolCtx := protocol.ProtocolBuffersByContent(context)
	allField := protocolCtx.GetReqHeaders()
	//serialize header
	//headerMap := sofarpc.GetMap(context, defaultTmpBufferSize)
	//headerMap := buffers.HeaderMap

	//logger
	logger := log.ByContext(context)

	serializeIns.DeSerialize(requestCommand.HeaderMap, &allField)
	logger.Debugf("deserialize header map:%v", allField)

	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(requestCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(requestCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(requestCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(requestCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(requestCommand.ReqID), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(requestCommand.CodecPro), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)] = strconv.FormatUint(uint64(requestCommand.Timeout), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(requestCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(requestCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(requestCommand.ContentLen), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(requestCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = className
	logger.Debugf("request class name is:%s", className)

	//sofarpc.ReleaseMap(context, headerMap)

	requestCommand.RequestHeader = allField
}

func deserializeRequestAllFieldsV2(context context.Context, requestCommandV2 *sofarpc.BoltV2RequestCommand) {
	deserializeRequestAllFields(context, &requestCommandV2.BoltRequestCommand)
	requestCommandV2.RequestHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(requestCommandV2.Version1), 10)
	requestCommandV2.RequestHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(requestCommandV2.SwitchCode), 10)
}
