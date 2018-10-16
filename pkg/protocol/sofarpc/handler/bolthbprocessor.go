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

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type boltHbProcessor struct {
}

func (b *boltHbProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	logger := log.ByContext(context)

	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequest(context, cmd)
		reqID := protocol.StreamIDConv(cmd.ReqID)

		//Heartbeat message only has request header
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				status := filter.OnDecodeHeader(reqID, cmd, true)
				//		logger.Debugf("Process Heartbeat Request Msg")

				if status == types.Stop {
					return
				}
			}
		}
	} else if cmd, ok := msg.(*sofarpc.BoltResponseCommand); ok {
		deserializeResponse(context, cmd)
		reqID := protocol.StreamIDConv(cmd.ReqID)
		//logger := log.ByContext(context)

		//Heartbeat message only has request header
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				status := filter.OnDecodeHeader(reqID, cmd, true)
				//	logger.Debugf("Process Heartbeat Response Msg")

				if status == types.Stop {
					return
				}
			}
		}
	} else {
		logger.Errorf("decode heart beat error\n")
	}
}
