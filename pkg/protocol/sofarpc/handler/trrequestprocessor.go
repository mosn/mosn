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

	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize/hessian"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type TrRequestProcessor struct{}

// ctx = type.serverStreamConnection
// CALLBACK STREAM LEVEL'S OnDecodeHeaders
func (b *TrRequestProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.TrRequestCommand); ok {
		deserializeRequestAllFieldsTR(cmd, context)
		streamId := atomic.AddUint32(&streamIdCsounter, 1)
		streamIdStr := sofarpc.StreamIDConvert(streamId)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				status := filter.OnDecodeHeader(streamIdStr, cmd.RequestHeader)
				if status == types.StopIteration {
					return
				}
			}

			if cmd.RequestContent != nil {
				status := filter.OnDecodeData(streamIdStr, buffer.NewIoBufferBytes(cmd.RequestContent))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

//Convert TR's Protocol Header  and Content Header to Map[string]string
func deserializeRequestAllFieldsTR(requestCommand *sofarpc.TrRequestCommand, context context.Context) {

	//DeSerialize Hessian
	hessianSerialize := hessian.HessianInstance

	ConnRequstBytes := requestCommand.ConnClassContent
	AppRequstBytes := requestCommand.AppClassContent

	requestCommand.RequestID = hessianSerialize.SerializeConnRequestBytes(ConnRequstBytes)
	requestCommand.TargetServiceUniqueName = hessianSerialize.SerializeAppRequestBytes(AppRequstBytes)

	allField := sofarpc.GetMap(context, 20)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(requestCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqFlag)] = strconv.FormatUint(uint64(requestCommand.RequestFlag), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderSeriProtocol)] = strconv.FormatUint(uint64(requestCommand.SerializeProtocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderDirection)] = strconv.FormatUint(uint64(requestCommand.Direction), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReserved)] = strconv.FormatUint(uint64(requestCommand.Reserved), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderAppclassnamelen)] = strconv.FormatUint(uint64(requestCommand.AppClassNameLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderConnrequestlen)] = strconv.FormatUint(uint64(requestCommand.ConnRequestLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderAppclasscontentlen)] = strconv.FormatUint(uint64(requestCommand.AppClassContentLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(requestCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(requestCommand.RequestID), 10)

	//TargetServiceUniqueName
	allField["service"] = requestCommand.TargetServiceUniqueName

	requestCommand.RequestHeader = allField
}
