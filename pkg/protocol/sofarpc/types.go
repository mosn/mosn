package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//bolt constants
const (
	//protocol code value
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
	RESPONSE_STATUS_SUCCESS int16 = 0 // 0x00
	RESPONSE_STATUS_ERROR int16 = 1 // 0x01
	RESPONSE_STATUS_SERVER_EXCEPTION int16 = 2 // 0x02
	RESPONSE_STATUS_UNKNOWN int16 = 3 // 0x03
	RESPONSE_STATUS_SERVER_THREADPOOL_BUSY int16 = 4 // 0x04
	RESPONSE_STATUS_ERROR_COMM int16 = 5 // 0x05
	RESPONSE_STATUS_NO_PROCESSOR int16 = 6 // 0x06
	RESPONSE_STATUS_TIMEOUT int16 = 7 // 0x07
	RESPONSE_STATUS_CLIENT_SEND_ERROR int16 = 8 // 0x08
	RESPONSE_STATUS_CODEC_EXCEPTION int16 = 9 // 0x09
	RESPONSE_STATUS_CONNECTION_CLOSED int16 = 16 // 0x10
	RESPONSE_STATUS_SERVER_SERIAL_EXCEPTION int16 = 17 // 0x11
	RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION int16 = 18 // 0x12
)

//tr constants
const (
	PROTOCOL_CODE       byte = 13
	PROTOCOL_HEADER_LEN int  = 14
)

//统一的RPC PROTOCOL抽象接口
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

type Protocols interface {
	Encode(value interface{}, data types.IoBuffer)

	Decode(ctx interface{}, data types.IoBuffer, out interface{})

	Handle(protocolCode byte, ctx interface{}, msg interface{})

	PutProtocol(protocolCode byte, protocol Protocol)

	GetProtocol(protocolCode byte) Protocol

	RegisterProtocol(protocolCode byte, protocol Protocol)

	UnRegisterProtocol(protocolCode byte)
}

//TODO
type HeartbeatTrigger interface {
	HeartbeatTriggered()
}

//TODO
type CommandHandler interface {
	HandleCommand(ctx interface{}, msg interface{})
	RegisterProcessor(cmdCode int16, processor *RemotingProcessor)

	//TODO executor selection
	//RegisterDefaultExecutor()
	//GetDefaultExecutor()
}

type RemotingProcessor interface {
	Process(ctx interface{}, msg interface{}, executor interface{})
}

type ProtoBasicCmd interface {
	GetProtocol() byte
	GetCmdCode() int16
	GetReqId() uint32
}

type BoltRequestCommand struct {
	Protocol byte  //BoltV1:1, BoltV2:2, Tr:13
	CmdType  byte  //Req:1,    Resp:0,   OneWay:2
	CmdCode  int16 //HB:0,     Req:1,    Resp:2
	Version  byte
	ReqId    uint32
	CodecPro byte

	Timeout int

	ClassLen      int16
	HeaderLen     int16
	ContentLen    int
	ClassName     []byte
	HeaderMap     []byte
	Content       []byte
	InvokeContext interface{}

	RequestHeader map[string]string
}

type BoltResponseCommand struct {
	Protocol byte  //BoltV1:1, BoltV2:2, Tr:13
	CmdType  byte  //Req:1,    Resp:0,   OneWay:2
	CmdCode  int16 //HB:0,     Req:1,    Resp:2
	Version  byte
	ReqId    uint32
	CodecPro byte

	ResponseStatus int16

	ClassLen      int16
	HeaderLen     int16
	ContentLen    int
	ClassName     []byte
	HeaderMap     []byte
	Content       []byte
	InvokeContext interface{}

	ResponseTimeMillis int64 //ResponseTimeMillis is not the field of the header
	ResponseHeader     map[string]string
}

type BoltV2RequestCommand struct {
	BoltRequestCommand
	Version1   byte
	SwitchCode byte
}

type BoltV2ResponseCommand struct {
	BoltResponseCommand
	Version1   byte
	SwitchCode byte
}

func (b *BoltRequestCommand) GetProtocol() byte {
	return b.Protocol
}

func (b *BoltRequestCommand) GetCmdCode() int16 {
	return b.CmdCode
}

func (b *BoltRequestCommand) GetReqId() uint32 {
	return b.ReqId
}

func (b *BoltResponseCommand) GetProtocol() byte {
	return b.Protocol
}

func (b *BoltResponseCommand) GetCmdCode() int16 {
	return b.CmdCode
}

func (b *BoltResponseCommand) GetReqId() uint32 {
	return b.ReqId
}

const (
	SofaRpcPropertyHeaderPrefix = "x-mosn-sofarpc-headers-property-"
)
