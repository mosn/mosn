package bolt

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

// Request is the cmd struct of bolt v1 request
type Request struct {
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
	Content    []byte

	RequestClass  string // deserialize fields
	RequestHeader map[string]string
}

// ~ RpcCmd
func (b *Request) ProtocolCode() byte {
	return b.Protocol
}

func (b *Request) RequestID() uint32 {
	return b.ReqID
}

func (b *Request) Header() map[string]string {
	return b.RequestHeader
}

func (b *Request) Data() []byte {
	return b.Content
}

func (b *Request) SetRequestID(requestID uint32) {
	b.ReqID = requestID
}

func (b *Request) SetHeader(header map[string]string) {
	b.RequestHeader = header
}

func (b *Request) SetData(data []byte) {
	b.Content = data
}

// ~ SofaRpcCmd
func (b *Request) CommandType() byte {
	return b.CmdType
}

func (b *Request) CommandCode() int16 {
	return b.CmdCode
}

// ~ HeaderMap
func (b *Request) Get(key string) (value string, ok bool) {
	value, ok = b.RequestHeader[key]
	return
}

func (b *Request) Set(key string, value string) {
	b.RequestHeader[key] = value
}

func (b *Request) Del(key string) {
	delete(b.RequestHeader, key)
}

func (b *Request) Range(f func(key, value string) bool) {
	for k, v := range b.RequestHeader {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

// Response is the cmd struct of bolt v1 response
type Response struct {
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
	Content    []byte

	ResponseClass  string // deserialize fields
	ResponseHeader map[string]string

	ResponseTimeMillis int64 //ResponseTimeMillis is not the field of the header
}

// ~ RpcCmd
func (b *Response) ProtocolCode() byte {
	return b.Protocol
}

func (b *Response) RequestID() uint32 {
	return b.ReqID
}

func (b *Response) Header() map[string]string {
	return b.ResponseHeader
}

func (b *Response) Data() []byte {
	return b.Content
}

func (b *Response) SetRequestID(requestID uint32) {
	b.ReqID = requestID
}

func (b *Response) SetHeader(header map[string]string) {
	b.ResponseHeader = header
}

func (b *Response) SetData(data []byte) {
	b.Content = data
}

// ~ SofaRpcCmd
func (b *Response) CommandType() byte {
	return b.CmdType
}

func (b *Response) CommandCode() int16 {
	return b.CmdCode
}

// ~ HeaderMap
func (b *Response) Get(key string) (value string, ok bool) {
	value, ok = b.ResponseHeader[key]
	return
}

func (b *Response) Set(key string, value string) {
	b.ResponseHeader[key] = value
}

func (b *Response) Del(key string) {
	delete(b.ResponseHeader, key)
}

func (b *Response) Range(f func(key, value string) bool) {
	for k, v := range b.ResponseHeader {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
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

// Request is the cmd struct of bolt v2 request
type RequestV2 struct {
	Request
	Version1   byte //00
	SwitchCode byte
}

// Response is the cmd struct of bolt v2 response
type ResponseV2 struct {
	Response
	Version1   byte //00
	SwitchCode byte
}
