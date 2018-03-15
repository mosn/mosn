package codec

import (
	"net"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
)

//command defination
type boltCommand struct {
	cmdCode int16
	version byte
	cmdType byte
	codec   byte
	//protoSwitchStatus byte

	id            uint32
	classLength   int16
	headerLength  int16
	contentLength int

	class   []byte
	header  []byte
	content []byte

	invokeContext interface{}
}

// bolt request command
type boltRequestCommand struct {
	//rpc command
	boltCommand

	//request command
	timeout int

	//rpc request command
	requestObject interface{}
	requestClass  string

	//customSerializer CustomSerializer
	requestHeader map[string]string
	arriveTime    int64
}

// bolt response command
type boltResponseCommand struct {
	//rpc command
	boltCommand

	//response command
	responseStatus     int16
	responseTimeMillis int64
	responseHost       net.Addr
	cause              error

	//rpc response command
	responseObject interface{}
	responseClass  string

	//customSerializer CustomSerializer
	responseHeader map[string]string

	errorMsg string
}

func (b *boltCommand) GetProtocolCode() byte {
	return sofarpc.PROTOCOL_CODE_V1
}

func (b *boltCommand) GetCmdCode() int16 {
	return b.cmdCode
}

func (b *boltCommand) GetId() uint32 {
	return b.id
}

func (b *boltCommand) GetClass() []byte {
	return b.class
}

func (b *boltCommand) GetHeader() []byte {
	return b.header
}

func (b *boltCommand) GetContent() []byte {
	return b.content
}

func (b *boltRequestCommand) GetProtocolCode() byte {
	return sofarpc.PROTOCOL_CODE_V1
}

func (b *boltRequestCommand) GetCmdCode() int16 {
	return b.cmdCode
}

func (b *boltRequestCommand) GetId() uint32 {
	return b.id
}

func (b *boltRequestCommand) SetTimeout(timeout int) {
	b.timeout = timeout
}

func (b *boltRequestCommand) SetArriveTime(arriveTime int64) {
	b.arriveTime = arriveTime
}

func (b *boltRequestCommand) SetRequestHeader(headerMap map[string]string) {
	b.requestHeader = headerMap
}

func (b *boltRequestCommand) GetRequestHeader() map[string]string {
	return b.requestHeader
}

func (b *boltResponseCommand) GetProtocolCode() byte {
	return sofarpc.PROTOCOL_CODE_V1
}

func (b *boltResponseCommand) GetCmdCode() int16 {
	return b.cmdCode
}

func (b *boltResponseCommand) GetId() uint32 {
	return b.id
}

func (b *boltResponseCommand) SetResponseStatus(status int16) {
	b.responseStatus = status
}

func (b *boltResponseCommand) SetResponseTimeMillis(responseTime int64) {
	b.responseTimeMillis = responseTime
}

func (b *boltResponseCommand) SetResponseHost(host net.Addr) {
	b.responseHost = host
}
