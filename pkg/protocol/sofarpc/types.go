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
	"context"
	"errors"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

//bolt constants used to
const (
	HeaderProtocolCode       string = "protocol"
	HeaderCmdType            string = "cmdtype"
	HeaderCmdCode            string = "cmdcode"
	HeaderVersion            string = "version"
	HeaderReqID              string = "requestid"
	HeaderCodec              string = "codec"
	HeaderTimeout            string = "timeout"
	HeaderClassLen           string = "classlen"
	HeaderHeaderLen          string = "headerlen"
	HeaderContentLen         string = "contentlen"
	HeaderClassName          string = "classname"
	HeaderVersion1           string = "ver1"
	HeaderSwitchCode         string = "switchcode"
	HeaderRespStatus         string = "respstatus"
	HeaderRespTimeMills      string = "resptimemills"
	HeaderReqFlag            string = "requestflag"
	HeaderSeriProtocol       string = "serializeprotocol"
	HeaderDirection          string = "direction"
	HeaderReserved           string = "reserved"
	HeaderAppclassnamelen    string = "appclassnamelen"
	HeaderConnrequestlen     string = "connrequestlen"
	HeaderAppclasscontentlen string = "appclasscontentlen"
)

// Encode/Decode Exception Msg
const (
	InvalidCommandType string = "Invalid command type for encoding"
	NoProCodeInHeader  string = "Protocol code not found in header when encoding"
	InvalidHeaderType  string = "Invalid header type, neither map nor command"
	UnKnownReqtype     string = "Unknown request type"
	UnKnownCmdcode     string = "Unknown cmd code"
	NoReqIDFound       string = "No request Id found in header"
	UnKnownCmd         string = "Unknown Command"
)

// ProtocolType used to define protocol code
type ProtocolType byte

// Define protocol code
const (
	BOLT_V1 ProtocolType = 1
	BOLT_V2 ProtocolType = 2
	SOFA_TR ProtocolType = 13
)

// protocol code value
const (
	PROTOCOL_CODE_V1 byte = 1
	PROTOCOL_CODE_V2 byte = 2

	PROTOCOL_VERSION_1 byte = 1
	PROTOCOL_VERSION_2 byte = 2

	REQUEST_HEADER_LEN_V1 int = 22
	REQUEST_HEADER_LEN_V2 int = 24

	RESPONSE_HEADER_LEN_V1 int = 20
	RESPONSE_HEADER_LEN_V2 int = 22

	LESS_LEN_V1 int = RESPONSE_HEADER_LEN_V1
	LESS_LEN_V2 int = RESPONSE_HEADER_LEN_V2

	RESPONSE       byte = 0
	REQUEST        byte = 1
	REQUEST_ONEWAY byte = 2

	//command code value
	HEARTBEAT    int16 = 0
	RPC_REQUEST  int16 = 1
	RPC_RESPONSE int16 = 2
	//response status
	RESPONSE_STATUS_SUCCESS                   int16 = 0  // 0x00
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

// Protocol's interface for getting xcode
type Protocol interface {
	/**
	 * Get the encoder for the protocol.
	 *
	 * @return
	 */
	GetEncoder() types.Encoder

	/**
	 * Get the decoder for the protocol.
	 *
	 * @return
	 */
	GetDecoder() types.Decoder

	/**
	 * Get the heartbeat trigger for the protocol.
	 *
	 * @return
	 */
	//TODO
	//GetHeartbeatTrigger() HeartbeatTrigger

	/**
	 * Get the command handler for the protocol.
	 *
	 * @return
	 */
	//TODO
	GetCommandHandler() CommandHandler
}

// CommandHandler for protocol to handle msg
type CommandHandler interface {
	HandleCommand(context context.Context, msg interface{}, filter interface{}) error

	RegisterProcessor(cmdCode int16, processor *RemotingProcessor)

	//TODO executor selection
	//RegisterDefaultExecutor()
	//GetDefaultExecutor()
}

// RemotingProcessor to process msg
type RemotingProcessor interface {
	Process(context context.Context, msg interface{}, filter interface{})
}

// ProtoBasicCmd act as basic cmd for many protocol
type ProtoBasicCmd interface {
	types.HeaderMap

	GetProtocol() byte

	GetCmdType() byte

	GetCmdCode() int16

	GetReqID() uint32

	GetRespStatus() uint32

	// TODO remove this method
	SetReqID(uint32)
}

// BoltRequestCommand is the cmd struct of bolt v1 request
type BoltRequestCommand struct {
	Protocol   byte  //BoltV1:1, BoltV2:2
	CmdType    byte  //Req:1,    Resp:0,   OneWay:2
	CmdCode    int16 //HB:0,     Req:1,    Resp:2
	Version    byte  //1
	ReqID      uint32
	CodecPro   byte
	Timeout    int
	ClassLen   int16
	HeaderLen  int16
	ContentLen int
	ClassName  []byte
	HeaderMap  []byte
	Content    []byte
	//InvokeContext interface{}

	RequestHeader map[string]string
	RequestClass  string
}

// BoltResponseCommand is the cmd struct of bolt v1 response
type BoltResponseCommand struct {
	Protocol byte  //BoltV1:1, BoltV2:2
	CmdType  byte  //Req:1,    Resp:0,   OneWay:2
	CmdCode  int16 //HB:0,     Req:1,    Resp:2
	Version  byte  //BoltV1:1  BoltV2: 1
	ReqID    uint32
	CodecPro byte // 1

	ResponseStatus int16 //Success:0 Error:1 Timeout:7

	ClassLen   int16
	HeaderLen  int16
	ContentLen int
	ClassName  []byte
	HeaderMap  []byte
	Content    []byte
	//InvokeContext interface{}

	ResponseTimeMillis int64 //ResponseTimeMillis is not the field of the header
	ResponseHeader     map[string]string
	ResponseClass      string
}

// BoltV2RequestCommand is the cmd struct of bolt v2 request
type BoltV2RequestCommand struct {
	BoltRequestCommand
	Version1   byte //00
	SwitchCode byte
}

// BoltV2RequestCommand is the cmd struct of bolt v2 response
type BoltV2ResponseCommand struct {
	BoltResponseCommand
	Version1   byte //00
	SwitchCode byte
}

// GetProtocol return BoltRequestCommand.protocol
func (b *BoltRequestCommand) GetProtocol() byte {
	return b.Protocol
}

// GetCmdType return BoltRequestCommand.CmdType
func (b *BoltRequestCommand) GetCmdType() byte {
	return b.CmdType
}

// GetCmdCode return BoltRequestCommand.CmdCode
func (b *BoltRequestCommand) GetCmdCode() int16 {
	return b.CmdCode
}

// GetReqID return BoltRequestCommand.ReqID
func (b *BoltRequestCommand) GetReqID() uint32 {
	return b.ReqID
}

// GetRespStatus return RESPONSE_STATUS_SUCCESS for request command
func (b *BoltRequestCommand) GetRespStatus() uint32 {
	return uint32(RESPONSE_STATUS_SUCCESS)
}

func (b *BoltRequestCommand) SetReqID(reqId uint32) {
	b.ReqID = reqId
}

// GetProtocol return BoltResponseCommand.Protocol
func (b *BoltResponseCommand) GetProtocol() byte {
	return b.Protocol
}

// GetProtocol return BoltResponseCommand.CmdType
func (b *BoltResponseCommand) GetCmdType() byte {
	return b.CmdType
}

// GetCmdCode return BoltResponseCommand.CmdCode
func (b *BoltResponseCommand) GetCmdCode() int16 {
	return b.CmdCode
}

// GetReqID return BoltResponseCommand.ReqID
func (b *BoltResponseCommand) GetReqID() uint32 {
	return b.ReqID
}

// GetRespStatus return BoltResponseCommand.ResponseStatus
func (b *BoltResponseCommand) GetRespStatus() uint32 {
	return uint32(b.ResponseStatus)
}

func (b *BoltResponseCommand) SetReqID(reqId uint32) {
	b.ReqID = reqId
}

// ~ HeaderMap
func (b *BoltRequestCommand) Get(key string) (value string, ok bool) {
	value, ok = b.RequestHeader[key]
	return
}

func (b *BoltRequestCommand) Set(key string, value string) {
	b.RequestHeader[key] = value
}

func (b *BoltRequestCommand) Del(key string) {
	delete(b.RequestHeader, key)
}

func (b *BoltRequestCommand) Range(f func(key, value string) bool) {
	for k, v := range b.RequestHeader {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

func (b *BoltRequestCommand) ByteSize() uint64 {
	var size uint64

	for k, v := range b.RequestHeader {
		size += uint64(len(k) + len(v))
	}
	return size
}

// ~ HeaderMap
func (b *BoltResponseCommand) Get(key string) (value string, ok bool) {
	value, ok = b.ResponseHeader[key]
	return
}

func (b *BoltResponseCommand) Set(key string, value string) {
	b.ResponseHeader[key] = value
}

func (b *BoltResponseCommand) Del(key string) {
	delete(b.ResponseHeader, key)
}

func (b *BoltResponseCommand) Range(f func(key, value string) bool) {
	for k, v := range b.ResponseHeader {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

func (b *BoltResponseCommand) ByteSize() uint64 {
	var size uint64

	for k, v := range b.ResponseHeader {
		size += uint64(len(k) + len(v))
	}
	return size
}

// mosn sofarpc headers' namespace
const (
	SofaRPCPropertyHeaderPrefix = "x-mosn-sofarpc-headers-property-"
)

const (
	HEADER_REQUEST         byte   = 0
	HEADER_RESPONSE        byte   = 1
	HESSIAN_SERIALIZE      byte   = 1
	JAVA_SERIALIZE         byte   = 2
	TOP_SERIALIZE          byte   = 3
	HESSIAN2_SERIALIZE     byte   = 4
	HEADER_ONEWAY          byte   = 1
	HEADER_TWOWAY          byte   = 2
	PROCOCOL_VERSION       byte   = 13
	PROTOCOL_HEADER_LENGTH uint32 = 14
)

// BuildSofaRespMsg build sofa response msg according to headers and respStatus
func BuildSofaRespMsg(ctx context.Context, cmd ProtoBasicCmd, respStatus int16) (ProtoBasicCmd, error) {
	switch c := cmd.(type) {
	case *BoltRequestCommand:
		return &BoltResponseCommand{
			Protocol:       c.Protocol,
			CmdType:        RESPONSE,
			CmdCode:        RPC_RESPONSE,
			Version:        c.Version,
			ReqID:          c.ReqID,
			CodecPro:       c.CodecPro,
			ResponseStatus: respStatus,
		}, nil

	case *BoltV2RequestCommand:
		return &BoltV2ResponseCommand{
			BoltResponseCommand: BoltResponseCommand{
				Protocol:       c.Protocol,
				CmdType:        RESPONSE,
				CmdCode:        RPC_RESPONSE,
				Version:        c.Version,
				ReqID:          c.ReqID,
				CodecPro:       c.CodecPro,
				ResponseStatus: respStatus,
			},
			Version1:   c.Version1,
			SwitchCode: c.SwitchCode,
		}, nil
	}

	log.ByContext(ctx).Errorf("[BuildSofaRespMsg Error]Unknown cmd type")
	return cmd, errors.New(types.UnSupportedProCode)
}

// Sofa Rpc Default HC Parameters
const (
	SofaRPC                             = "SofaRpc"
	DefaultBoltHeartBeatTimeout         = 6 * 15 * time.Second
	DefaultBoltHeartBeatInterval        = 15 * time.Second
	DefaultIntervalJitter               = 5 * time.Millisecond
	DefaultHealthyThreshold      uint32 = 2
	DefaultUnhealthyThreshold    uint32 = 2
)

// DefaultSofaRPCHealthCheckConf
var DefaultSofaRPCHealthCheckConf = v2.HealthCheck{
	HealthCheckConfig: v2.HealthCheckConfig{
		Protocol:           SofaRPC,
		HealthyThreshold:   DefaultHealthyThreshold,
		UnhealthyThreshold: DefaultUnhealthyThreshold,
	},
	Timeout:        DefaultBoltHeartBeatTimeout,
	Interval:       DefaultBoltHeartBeatInterval,
	IntervalJitter: DefaultIntervalJitter,
}
