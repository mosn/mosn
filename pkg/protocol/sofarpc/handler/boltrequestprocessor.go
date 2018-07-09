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
	"sync/atomic"

	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/models"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var streamIdCounter uint32
var defaultTmpBufferSize = 1 << 6

type BoltRequestProcessor struct{}

type BoltRequestProcessorV2 struct{}

// ctx = type.serverStreamConnection
// CALLBACK STREAM LEVEL'S OnReceiveHeaders
func (b *BoltRequestProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(context, cmd)
		streamId := atomic.AddUint32(&streamIdCounter, 1)
		streamIdStr := sofarpc.StreamIDConvert(streamId)

		//print tracer log
		log.DefaultLogger.Debugf("time=%s,tracerId=%s,streamId=%s,protocol=%s,service=%s,callerIp=%s", time.Now(), cmd.RequestHeader[models.TRACER_ID_KEY], streamIdStr, cmd.RequestHeader[models.SERVICE_KEY], "bolt", cmd.RequestHeader[models.CALLER_IP_KEY])

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				if cmd.Content == nil {
					cmd.RequestHeader[types.HeaderStremEnd] = "yes"
				}
				
				status := filter.OnDecodeHeader(streamIdStr, cmd.RequestHeader)
				
				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamIdStr, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

// ctx = type.serverStreamConnection
func (b *BoltRequestProcessorV2) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltV2RequestCommand); ok {
		deserializeRequestAllFieldsV2(cmd, context)
		streamId := atomic.AddUint32(&streamIdCounter, 1)
		streamIdStr := sofarpc.StreamIDConvert(streamId)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				
				if cmd.Content == nil {
					cmd.RequestHeader["x-mosn-endstream"] = "yes"
				}
				
				status := filter.OnDecodeHeader(streamIdStr, cmd.RequestHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamIdStr, buffer.NewIoBufferBytes(cmd.Content))

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

	//serialize header
	headerMap := sofarpc.GetMap(context, defaultTmpBufferSize)

	//logger
	logger := log.ByContext(context)

	serializeIns.DeSerialize(requestCommand.HeaderMap, &headerMap)
	logger.Debugf("deserialize header map:%v", headerMap)

	allField := sofarpc.GetMap(context, 20+len(headerMap))
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(requestCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(requestCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(requestCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(requestCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(requestCommand.ReqId), 10)
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

	for k, v := range headerMap {
		allField[k] = v
	}

	sofarpc.ReleaseMap(context, headerMap)

	requestCommand.RequestHeader = allField
}

func deserializeRequestAllFieldsV2(requestCommandV2 *sofarpc.BoltV2RequestCommand, context context.Context) {
	deserializeRequestAllFields(context, &requestCommandV2.BoltRequestCommand)
	requestCommandV2.RequestHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(requestCommandV2.Version1), 10)
	requestCommandV2.RequestHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(requestCommandV2.SwitchCode), 10)
}
