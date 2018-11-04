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
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/models"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// BoltRequestProcessor process bolt v1 request msg
type BoltRequestProcessor struct{}

// BoltRequestProcessorV2 process bolt v2 request msg
type BoltRequestProcessorV2 struct{}

// Process boltv1 request cmd to map[string]string header
func (b *BoltRequestProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequest(context, cmd)
		streamID := protocol.GenerateIDString()
		//print tracer log
		log.DefaultLogger.Debugf("time=%s,tracerID=%s,streamID=%s,protocol=%s,service=%s,callerIp=%s", time.Now(), cmd.RequestHeader[models.TRACER_ID_KEY], streamID, "bolt", cmd.RequestHeader[models.SERVICE_KEY], cmd.RequestHeader[models.CALLER_IP_KEY])

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {

				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				status := filter.OnDecodeHeader(streamID, cmd, cmd.Content == nil)

				if status == types.Stop {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamID, buffer.NewIoBufferBytes(cmd.Content), true)

				if status == types.Stop {
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
		//deserializeRequestAllFieldsV2(context, cmd)
		deserializeRequest(context, &cmd.BoltRequestCommand)
		streamID := protocol.GenerateIDString()

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				status := filter.OnDecodeHeader(streamID, cmd, cmd.Content == nil)

				if status == types.Stop {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamID, buffer.NewIoBufferBytes(cmd.Content), true)

				if status == types.Stop {
					return
				}
			}
		}
	}
}

func deserializeRequest(context context.Context, requestCommand *sofarpc.BoltRequestCommand) {
	//get instance
	serializeIns := serialize.Instance

	protocolCtx := protocol.ProtocolBuffersByContext(context)
	requestCommand.RequestHeader = protocolCtx.GetReqHeaders()

	//logger
	logger := log.ByContext(context)

	//serialize header
	serializeIns.DeSerialize(requestCommand.HeaderMap, &requestCommand.RequestHeader)
	logger.Debugf("Deserialize request header map:%v", requestCommand.RequestHeader)

	//serialize class name
	serializeIns.DeSerialize(requestCommand.ClassName, &requestCommand.RequestClass)
	logger.Debugf("Request class name is:%s", requestCommand.RequestClass)
}
