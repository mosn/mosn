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

package sofarpc

import (
	"errors"
	"time"

	apiv2 "github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// SofaRpcCmd  act as basic model for sofa protocols
type SofaRpcCmd interface {
	rpc.RpcCmd

	// CommandType returns the command type, request/request_oneway/response
	CommandType() byte

	// CommandCode return the command code, rpc_request/rpc_response/heartbeat
	CommandCode() int16
}

// HeartbeatBuilder provides interface to construct proper heartbeat command for sofarpc sub-protocols
type HeartbeatBuilder interface {
	// Trigger builds an active heartbeat command
	Trigger() SofaRpcCmd

	// Reply builds heartbeat command corresponding to the given requestID
	Reply() SofaRpcCmd
}

// HeartbeatBuilder provides interface to construct proper response command for sofarpc sub-protocols
type ResponseBuilder interface {
	// BuildResponse build response with given status code
	BuildResponse(status int16) SofaRpcCmd
}

// bolt constants
const (
	// ~~ header name of protocol field
	HeaderProtocolCode  string = "protocol"
	HeaderCmdType       string = "cmdtype"
	HeaderCmdCode       string = "cmdcode"
	HeaderVersion       string = "version"
	HeaderReqID         string = "requestid"
	HeaderCodec         string = "codec"
	HeaderTimeout       string = "timeout"
	HeaderClassLen      string = "classlen"
	HeaderHeaderLen     string = "headerlen"
	HeaderContentLen    string = "contentlen"
	HeaderClassName     string = "classname"
	HeaderVersion1      string = "ver1"
	HeaderSwitchCode    string = "switchcode"
	HeaderRespStatus    string = "respstatus"
	HeaderRespTimeMills string = "resptimemills"

	// ~~ constans
	PROTOCOL_CODE_V1 byte = 1 // protocol code
	PROTOCOL_CODE_V2 byte = 2

	PROTOCOL_VERSION_1 byte = 1 // version
	PROTOCOL_VERSION_2 byte = 2

	REQUEST_HEADER_LEN_V1 int = 22 // protocol header fields length
	REQUEST_HEADER_LEN_V2 int = 24

	RESPONSE_HEADER_LEN_V1 int = 20
	RESPONSE_HEADER_LEN_V2 int = 22

	LESS_LEN_V1 int = RESPONSE_HEADER_LEN_V1 // minimal length for decoding
	LESS_LEN_V2 int = RESPONSE_HEADER_LEN_V2

	RESPONSE       byte = 0 // cmd type
	REQUEST        byte = 1
	REQUEST_ONEWAY byte = 2

	HEARTBEAT    int16 = 0 // cmd code
	RPC_REQUEST  int16 = 1
	RPC_RESPONSE int16 = 2

	HESSIAN2_SERIALIZE byte = 1 // serialize

	RESPONSE_STATUS_SUCCESS                   int16 = 0  // 0x00 response status
	RESPONSE_STATUS_ERROR                     int16 = 1  // 0x01
	RESPONSE_STATUS_SERVER_EXCEPTION          int16 = 2  // 0x02
	RESPONSE_STATUS_UNKNOWN                   int16 = 3  // 0x03
	RESPONSE_STATUS_SERVER_THREADPOOL_BUSY    int16 = 4  // 0x04
	RESPONSE_STATUS_ERROR_COMM                int16 = 5  // 0x05
	RESPONSE_STATUS_NO_PROCESSOR              int16 = 6  // 0x06
	RESPONSE_STATUS_TIMEOUT                   int16 = 7  // 0x07
	RESPONSE_STATUS_CLIENT_SEND_ERROR         int16 = 8  // 0x08
	RESPONSE_STATUS_CODEC_EXCEPTION           int16 = 9  // 0x09
	RESPONSE_STATUS_CONNECTION_CLOSED         int16 = 16 // 0x10
	RESPONSE_STATUS_SERVER_SERIAL_EXCEPTION   int16 = 17 // 0x11
	RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION int16 = 18 // 0x12
)

const (
	// Encode/Decode Exception Msg
	UnKnownCmdType string = "unknown cmd type"
	UnKnownCmdCode string = "unknown cmd code"

	// Sofa Rpc Default HC Parameters
	SofaRPC                             = "SofaRpc"
	DefaultBoltHeartBeatTimeout         = 6 * 15 * time.Second
	DefaultBoltHeartBeatInterval        = 15 * time.Second
	DefaultIntervalJitter               = 5 * time.Millisecond
	DefaultHealthyThreshold      uint32 = 2
	DefaultUnhealthyThreshold    uint32 = 2
)

var (
	// Encode/Decode Exception
	ErrUnKnownCmdType = errors.New(UnKnownCmdType)
	ErrUnKnownCmdCode = errors.New(UnKnownCmdCode)
)

// DefaultSofaRPCHealthCheckConf
var DefaultSofaRPCHealthCheckConf = apiv2.HealthCheck{
	HealthCheckConfig: apiv2.HealthCheckConfig{
		Protocol:           SofaRPC,
		HealthyThreshold:   DefaultHealthyThreshold,
		UnhealthyThreshold: DefaultUnhealthyThreshold,
	},
	Timeout:        DefaultBoltHeartBeatTimeout,
	Interval:       DefaultBoltHeartBeatInterval,
	IntervalJitter: DefaultIntervalJitter,
}

// ~~ command definitions

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

const RequestHeaderLenIndex = 16
const ResponseHeaderLenIndex = 14

// BoltRequest is the cmd struct of bolt v1 request
type BoltRequest struct {
	Protocol byte  //BoltV1:1, BoltV2:2
	CmdType  byte  //Req:1,    Resp:0,   OneWay:2
	CmdCode  int16 //HB:0,     Req:1,    Resp:2
	Version  byte  //1
	ReqID    uint32
	Codec    byte

	Timeout int

	ClassLen   int16
	HeaderLen  int16
	ContentLen int
	ClassName  []byte
	HeaderMap  []byte
	Content    types.IoBuffer

	RequestClass  string // deserialize fields
	RequestHeader map[string]string
}

// ~ RpcCmd
func (b *BoltRequest) ProtocolCode() byte {
	return b.Protocol
}

func (b *BoltRequest) RequestID() uint64 {
	return uint64(b.ReqID)
}

func (b *BoltRequest) Header() map[string]string {
	return b.RequestHeader
}

func (b *BoltRequest) Data() types.IoBuffer {
	return b.Content
}

func (b *BoltRequest) SetRequestID(requestID uint64) {
	b.ReqID = uint32(requestID)
}

func (b *BoltRequest) SetHeader(header map[string]string) {
	b.RequestHeader = header
}

func (b *BoltRequest) SetData(data types.IoBuffer) {
	b.Content = data
}

func (b *BoltRequest) GetTimeout() int {
	return b.Timeout
}

// ~ SofaRpcCmd
func (b *BoltRequest) CommandType() byte {
	return b.CmdType
}

func (b *BoltRequest) CommandCode() int16 {
	return b.CmdCode
}

// ~ HeaderMap
func (b *BoltRequest) Get(key string) (value string, ok bool) {
	value, ok = b.RequestHeader[key]
	return
}

func (b *BoltRequest) Set(key string, value string) {
	b.RequestHeader[key] = value
}

func (b *BoltRequest) Add(key string, value string) {
	panic("not supported")
}

func (b *BoltRequest) Del(key string) {
	delete(b.RequestHeader, key)
}

func (b *BoltRequest) Range(f func(key, value string) bool) {
	for k, v := range b.RequestHeader {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

func (b *BoltRequest) Clone() types.HeaderMap {
	copy := &BoltRequest{}
	*copy = *b

	// deep copy
	requestHeader := make(map[string]string, len(b.RequestHeader))
	for k, v := range b.RequestHeader {
		requestHeader[k] = v
	}
	copy.RequestHeader = requestHeader

	return copy
}

func (b *BoltRequest) ByteSize() (size uint64) {
	for k, v := range b.RequestHeader {
		size += uint64(len(k) + len(v))
	}
	return
}

// BoltResponse is the cmd struct of bolt v1 response
type BoltResponse struct {
	Protocol byte  //BoltV1:1, BoltV2:2
	CmdType  byte  //Req:1,    Resp:0,   OneWay:2
	CmdCode  int16 //HB:0,     Req:1,    Resp:2
	Version  byte  //BoltV1:1  BoltV2: 1
	ReqID    uint32
	Codec    byte // 1

	ResponseStatus int16 //Success:0 Error:1 Timeout:7

	ClassLen   int16
	HeaderLen  int16
	ContentLen int
	ClassName  []byte
	HeaderMap  []byte
	Content    types.IoBuffer

	ResponseClass  string // deserialize fields
	ResponseHeader map[string]string

	ResponseTimeMillis int64 //ResponseTimeMillis is not the field of the header
}

// ~ RpcCmd
func (b *BoltResponse) ProtocolCode() byte {
	return b.Protocol
}

func (b *BoltResponse) RequestID() uint64 {
	return uint64(b.ReqID)
}

func (b *BoltResponse) Header() map[string]string {
	return b.ResponseHeader
}

func (b *BoltResponse) Data() types.IoBuffer {
	return b.Content
}

func (b *BoltResponse) SetRequestID(requestID uint64) {
	b.ReqID = uint32(requestID)
}

func (b *BoltResponse) SetHeader(header map[string]string) {
	b.ResponseHeader = header
}

func (b *BoltResponse) SetData(data types.IoBuffer) {
	b.Content = data
}

// response have no timeout
func (b *BoltResponse) GetTimeout() int {
	return -1
}

// ~ ResponseStatus
func (b *BoltResponse) RespStatus() uint32 {
	return uint32(b.ResponseStatus)
}

// ~ SofaRpcCmd
func (b *BoltResponse) CommandType() byte {
	return b.CmdType
}

func (b *BoltResponse) CommandCode() int16 {
	return b.CmdCode
}

// ~ HeaderMap
func (b *BoltResponse) Get(key string) (value string, ok bool) {
	value, ok = b.ResponseHeader[key]
	return
}

func (b *BoltResponse) Set(key string, value string) {
	b.ResponseHeader[key] = value
}

func (b *BoltResponse) Add(key string, value string) {
	panic("not supported")
}

func (b *BoltResponse) Del(key string) {
	delete(b.ResponseHeader, key)
}

func (b *BoltResponse) Range(f func(key, value string) bool) {
	for k, v := range b.ResponseHeader {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

func (b *BoltResponse) Clone() types.HeaderMap {
	copy := &BoltResponse{}
	*copy = *b

	// deep copy
	responseHeader := make(map[string]string, len(b.ResponseHeader))
	for k, v := range b.ResponseHeader {
		responseHeader[k] = v
	}
	copy.ResponseHeader = responseHeader

	return copy
}

func (b *BoltResponse) ByteSize() (size uint64) {
	for k, v := range b.ResponseHeader {
		size += uint64(len(k) + len(v))
	}
	return
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

const RequestV2HeaderLenIndex = 18
const ResponseV2HeaderLenIndex = 16

// BoltRequestV2 is the cmd struct of bolt v2 request
type BoltRequestV2 struct {
	BoltRequest
	Version1   byte //00
	SwitchCode byte
}

// BoltResponseV2 is the cmd struct of bolt v2 response
type BoltResponseV2 struct {
	BoltResponse
	Version1   byte //00
	SwitchCode byte
}
