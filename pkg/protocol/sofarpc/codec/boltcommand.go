package codec

import (
	"net"
)

//command defination
type boltCommand struct {
	protocol byte  // 1 boltv1, 2 boltv2, 13 tr request, 14 tr response
	cmdType  byte  // 0 response,1 是request,2是 req oneway
	cmdCode  int16 // 0 心跳,1 request, 2 response
	version  byte  // ver2
	id       uint32
	codec    byte
	//protoSwitchStatus byte

	classLength   int16
	headerLength  int16
	contentLength int

	class   []byte //CLASS NAME
	header  []byte //HEADER
	content []byte //CONTENT BYTES

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

	//add ver1 and switchcode for boltv2 request
	ver1       byte
	switchCode byte
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

	//add ver1 and switchcode for boltv2 response
	ver1       byte
	switchCode byte
}

//GetCmdType()byte
//GetVersion()byte
//GetCodec()byte
//GetTimeout()int
//GetClassLength()int16
//GetHeaderLength()int16
//GetContentLength()int
func (b *boltCommand) GetProtocolCode() byte {
	//return sofarpc.PROTOCOL_CODE_V1
	return b.protocol
}

func (b *boltCommand) GetCmdType() byte {
	return b.cmdType
}

func (b *boltCommand) GetCmdCode() int16 {
	return b.cmdCode
}

func (b *boltCommand) GetVersion() byte {
	return b.version
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

func (b *boltCommand) GetClassLength() int16 {
	return b.classLength
}

func (b *boltCommand) GetHeaderLength() int16 {
	return b.classLength
}

func (b *boltCommand) GetContentLength() int {
	return b.contentLength
}

func (b *boltCommand) GetCodec() byte {
	return b.codec
}

func (b *boltRequestCommand) GetTimeout() int {
	return b.timeout
}
func (b *boltRequestCommand) SetTimeout(timeout int) {
	b.timeout = timeout
}

func (b *boltRequestCommand) SetArriveTime(arriveTime int64) {
	b.arriveTime = arriveTime
}
func (b *boltRequestCommand) GetArriveTime() int64 {
	return b.arriveTime
}

func (b *boltRequestCommand) SetRequestHeader(headerMap map[string]string) {
	b.requestHeader = headerMap
}

func (b *boltRequestCommand) GetRequestHeader() map[string]string {
	return b.requestHeader
}

// FOR BOLT V2 Request
func (b *boltRequestCommand) SetVer1(ver1 byte) {
	b.ver1 = ver1
}
func (b *boltRequestCommand) GetVer1() byte {
	return b.ver1
}

func (b *boltRequestCommand) SetSwitch(switchCode byte) {
	b.switchCode = switchCode
}
func (b *boltRequestCommand) GetSwitch() byte {
	return b.switchCode
}

//RESPONSE
func (b *boltResponseCommand) SetResponseStatus(status int16) {
	b.responseStatus = status
}

func (b *boltResponseCommand) SetResponseTimeMillis(responseTime int64) {
	b.responseTimeMillis = responseTime
}

func (b *boltResponseCommand) GetResponseTimeMillis() int64 {
	return b.responseTimeMillis
}

func (b *boltResponseCommand) SetResponseHost(host net.Addr) {
	b.responseHost = host
}

func (b *boltResponseCommand) GetResponseStatus() int16 {
	return b.responseStatus
}

func (b *boltResponseCommand) SetResponseHeader(headerMap map[string]string) {
	b.responseHeader = headerMap
}

func (b *boltResponseCommand) GetResponseHeader() map[string]string {
	return b.responseHeader
}

//FOR BOLT V2
func (b *boltResponseCommand) SetVer1(ver1 byte) {
	b.ver1 = ver1
}
func (b *boltResponseCommand) GetVer1() byte {
	return b.ver1
}

func (b *boltResponseCommand) SetSwitch(switchCode byte) {
	b.switchCode = switchCode
}
func (b *boltResponseCommand) GetSwitch() byte {
	return b.switchCode
}
