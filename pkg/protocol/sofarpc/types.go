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

type RpcCommand interface {
	GetProtocolCode() byte

	GetCmdCode() int16

	GetId() uint32
}

type BoltRequestCommand interface {
	GetProtocolCode() byte

	GetCmdType() byte

	GetCmdCode() int16

	GetVersion() byte

	GetId() uint32 //get request id

	GetCodec() byte

	GetTimeout() int

	GetClassLength() int16

	GetHeaderLength() int16

	GetContentLength() int

	GetClass() []byte

	GetHeader() []byte

	GetContent() []byte

	SetRequestHeader(headerMap map[string]string)

	GetRequestHeader() map[string]string
}

type BoltResponseCommand interface {
	GetProtocolCode() byte

	GetCmdType() byte

	GetCmdCode() int16

	GetVersion() byte

	GetId() uint32 //get request id

	GetCodec() byte

	GetClassLength() int16

	GetHeaderLength() int16

	GetContentLength() int

	GetClass() []byte

	GetHeader() []byte

	GetContent() []byte

	GetResponseStatus() int16
	GetResponseTimeMillis() int64

	SetResponseHeader(headerMap map[string]string)

	GetResponseHeader() map[string]string
}

type TrRequestCommand interface {
	GetCmdCode() int16
}

const (
	SofaRpcHeaderPrefix = "x-mosn-sofarpc-"
)
