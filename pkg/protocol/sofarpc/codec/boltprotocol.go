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

package codec

import (
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/handler"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	sofarpc.RegisterProtocol(sofarpc.PROTOCOL_CODE_V1, BoltV1)
	sofarpc.RegisterProtocol(sofarpc.PROTOCOL_CODE_V2, BoltV2)
}

/**
 * Request command protocol for v1
 * 0     1     2           4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |proto| type| cmdcode   |ver2 |   requestID           |codec|        timeout        |  classLen |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * |headerLen  | contentLen            |                             ... ...                       |
 * +-----------+-----------+-----------+                                                                                               +
 * |               className + header  + content  bytes                                            |
 * +                                                                                               +
 * |                               ... ...                                                         |
 * +-----------------------------------------------------------------------------------------------+
 *
 * proto: code for protocol
 * type: request/response/request oneway
 * cmdcode: code for remoting command
 * ver2:version for remoting command
 * requestID: id of request
 * codec: code for codec
 * headerLen: length of header
 * contentLen: length of content
 *
 * Response command protocol for v1
 * 0     1     2     3     4           6           8          10           12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
 * |proto| type| cmdcode   |ver2 |   requestID           |codec|respstatus |  classLen |headerLen  |
 * +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
 * | contentLen            |                  ... ...                                              |
 * +-----------------------+                                                                       +
 * |                         className + header  + content  bytes                                  |
 * +                                                                                               +
 * |                               ... ...                                                         |
 * +-----------------------------------------------------------------------------------------------+
 * respstatus: response status
 */

// BoltV1 is the instance of boltProtocol of boltv1
var BoltV1 = &boltProtocol{
	sofarpc.PROTOCOL_CODE_V1,
	sofarpc.REQUEST_HEADER_LEN_V1,
	sofarpc.RESPONSE_HEADER_LEN_V1,
	&boltV1Codec{},
	&boltV1Codec{},
	handler.NewBoltCommandHandler(),
}

/**
 * Request command protocol for v2
 * 0     1     2           4           6           8          10     11     12          14         16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+------+-----+-----+-----+-----+
 * |proto| ver1|type | cmdcode   |ver2 |   requestID           |codec|switch|   timeout             |
 * +-----------+-----------+-----------+-----------+-----------+------------+-----------+-----------+
 * |classLen   |headerLen  |contentLen             |           ...                                  |
 * +-----------+-----------+-----------+-----------+                                                +
 * |               className + header  + content  bytes                                             |
 * +                                                                                                +
 * |                               ... ...                                  | CRC32(optional)       |
 * +------------------------------------------------------------------------------------------------+
 *
 * proto: code for protocol
 * ver1: version for protocol
 * type: request/response/request oneway
 * cmdcode: code for remoting command
 * ver2:version for remoting command
 * requestID: id of request
 * codec: code for codec
 * switch: function switch for protocol
 * headerLen: length of header
 * contentLen: length of content
 * CRC32: CRC32 of the frame(Exists when ver1 > 1)
 *
 * Response command protocol for v2
 * 0     1     2     3     4           6           8          10     11    12          14          16
 * +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+------+-----+-----+-----+-----+
 * |proto| ver1| type| cmdcode   |ver2 |   requestID           |codec|switch|respstatus |  classLen |
 * +-----------+-----------+-----------+-----------+-----------+------------+-----------+-----------+
 * |headerLen  | contentLen            |                      ...                                   |
 * +-----------------------------------+                                                            +
 * |               className + header  + content  bytes                                             |
 * +                                                                                                +
 * |                               ... ...                                  | CRC32(optional)       |
 * +------------------------------------------------------------------------------------------------+
 * respstatus: response status
 */

// BoltV2 is the instance of boltProtocol of boltv2
var BoltV2 = &boltProtocol{
	sofarpc.PROTOCOL_CODE_V2,
	sofarpc.REQUEST_HEADER_LEN_V2,
	sofarpc.RESPONSE_HEADER_LEN_V2,
	&boltV2Codec{},
	&boltV2Codec{},
	handler.NewBoltCommandHandlerV2(),
}

// Bolt protocol class
type boltProtocol struct {
	protocolCode      byte
	requestHeaderLen  int
	responseHeaderLen int

	encoder types.Encoder
	decoder types.Decoder
	//heartbeatTrigger			protocol.HeartbeatTrigger todo
	commandHandler sofarpc.CommandHandler
}

func (b *boltProtocol) GetRequestHeaderLength() int {
	return b.requestHeaderLen
}

func (b *boltProtocol) GetResponseHeaderLength() int {
	return b.responseHeaderLen
}

func (b *boltProtocol) GetEncoder() types.Encoder {
	return b.encoder
}

func (b *boltProtocol) GetDecoder() types.Decoder {
	return b.decoder
}

func (b *boltProtocol) GetCommandHandler() sofarpc.CommandHandler {
	return b.commandHandler
}

// NewBoltHeartbeat
// New Bolt Heartbeat with requestID as input
func NewBoltHeartbeat(requestID uint32) *sofarpc.BoltRequestCommand {
	return &sofarpc.BoltRequestCommand{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.HEARTBEAT,
		Version:  1,
		ReqID:    requestID,
		CodecPro: sofarpc.HESSIAN_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}
}

// NewBoltHeartbeatAck
// New Bolt Heartbeat Ack with requestID as input
func NewBoltHeartbeatAck(requestID uint32) *sofarpc.BoltResponseCommand {
	return &sofarpc.BoltResponseCommand{
		Protocol:       sofarpc.PROTOCOL_CODE_V1,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.HEARTBEAT,
		Version:        1,
		ReqID:          requestID,
		CodecPro:       sofarpc.HESSIAN_SERIALIZE, //todo: read default codec from config
		ResponseStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
	}
}
