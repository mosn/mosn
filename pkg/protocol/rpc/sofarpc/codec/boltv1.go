package codec

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/buffer"
)

var (
	BoltCodec = &boltCodec{}
)

func init() {
	sofarpc.Register(sofarpc.PROTOCOL_CODE_V1, BoltCodec, BoltCodec)
}

// ~~ types.Encoder
// ~~ types.Decoder
type boltCodec struct{}

func (c *boltCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch cmd := model.(type) {
	case *sofarpc.BoltRequest:
		return encodeRequest(ctx, cmd)
	case *sofarpc.BoltResponse:
		return encodeResponse(ctx, cmd)
	default:
		log.ByContext(ctx).Errorf("unknown model : %+v", model)
		return nil, rpc.ErrUnknownType
	}
}

func encodeRequest(ctx context.Context, cmd *sofarpc.BoltRequest) (types.IoBuffer, error) {
	// serialize classname and header
	if cmd.RequestClass != "" {
		cmd.ClassName, _ = serialize.Instance.Serialize(cmd.RequestClass)
		cmd.ClassLen = int16(len(cmd.ClassName))
	}

	if cmd.RequestHeader != nil {
		cmd.HeaderMap, _ = serialize.Instance.Serialize(cmd.RequestHeader)
		cmd.HeaderLen = int16(len(cmd.HeaderMap))
	}

	var b [4]byte

	size := sofarpc.REQUEST_HEADER_LEN_V1 + int(cmd.ClassLen) + len(cmd.HeaderMap)
	//buf := sofarpc.GetBuffer(context, size)

	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	//buf := protocolCtx.GetReqHeader(size)
	buf := buffer.NewIoBuffer(size)

	b[0] = cmd.Protocol
	buf.Write(b[0:1])
	b[0] = cmd.CmdType
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.CmdCode))
	buf.Write(b[0:2])

	b[0] = cmd.Version
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ReqID))
	buf.Write(b[0:4])

	b[0] = cmd.Codec
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.Timeout))
	buf.Write(b[0:4])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ClassLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.HeaderLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ContentLen))
	buf.Write(b[0:4])

	if cmd.ClassLen > 0 {
		buf.Write(cmd.ClassName)
	}

	if cmd.HeaderLen > 0 {
		buf.Write(cmd.HeaderMap)
	}
	return buf, nil
}

func encodeResponse(ctx context.Context, cmd *sofarpc.BoltResponse) (types.IoBuffer, error) {
	// serialize classname and header
	if cmd.ResponseClass != "" {
		cmd.ClassName, _ = serialize.Instance.Serialize(cmd.ResponseClass)
		cmd.ClassLen = int16(len(cmd.ClassName))
	}

	if cmd.ResponseHeader != nil {
		cmd.HeaderMap, _ = serialize.Instance.Serialize(cmd.ResponseHeader)
		cmd.HeaderLen = int16(len(cmd.HeaderMap))
	}

	var b [4]byte
	// todo: reuse bytes @boqin
	size := sofarpc.RESPONSE_HEADER_LEN_V1 + int(cmd.ClassLen) + len(cmd.HeaderMap)
	//buf := sofarpc.GetBuffer(context, size)
	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	//buf := protocolCtx.GetRspHeader(size)
	buf := buffer.NewIoBuffer(size)

	b[0] = cmd.Protocol
	buf.Write(b[0:1])
	b[0] = cmd.CmdType
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.CmdCode))
	buf.Write(b[0:2])

	b[0] = cmd.Version
	buf.Write(b[0:1])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ReqID))
	buf.Write(b[0:4])

	b[0] = cmd.Codec
	buf.Write(b[0:1])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ResponseStatus))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.ClassLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint16(b[0:], uint16(cmd.HeaderLen))
	buf.Write(b[0:2])

	binary.BigEndian.PutUint32(b[0:], uint32(cmd.ContentLen))
	buf.Write(b[0:4])

	if cmd.ClassLen > 0 {
		buf.Write(cmd.ClassName)
	}

	if cmd.HeaderLen > 0 {
		buf.Write(cmd.HeaderMap)
	}
	return buf, nil
}

func (c *boltCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	readableBytes := data.Len()
	read := 0
	var cmd interface{}
	logger := log.ByContext(ctx)

	if readableBytes >= sofarpc.LESS_LEN_V1 {
		bytes := data.Bytes()
		cmdType := bytes[1]

		//1. request
		if cmdType == sofarpc.REQUEST || cmdType == sofarpc.REQUEST_ONEWAY {
			if readableBytes >= sofarpc.REQUEST_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])
				ver2 := bytes[4]
				requestID := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				timeout := binary.BigEndian.Uint32(bytes[10:14])
				classLen := binary.BigEndian.Uint16(bytes[14:16])
				headerLen := binary.BigEndian.Uint16(bytes[16:18])
				contentLen := binary.BigEndian.Uint32(bytes[18:22])

				read = sofarpc.REQUEST_HEADER_LEN_V1
				var class, header, content []byte

				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytes[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read : read+int(contentLen)]
						read += int(contentLen)
					}

					data.Drain(read)

				} else { // not enough data

					logger.Debugf("BoltV1 DECODE Request, no enough data for fully decode")
					return cmd, nil
				}

				//sofabuffers := sofarpc.SofaProtocolBuffersByContext(ctx)
				//request := &sofabuffers.BoltReq
				request := &sofarpc.BoltRequest{}
				request.Protocol = sofarpc.PROTOCOL_CODE_V1
				request.CmdType = cmdType
				request.CmdCode = int16(cmdCode)
				request.Version = ver2
				request.ReqID = requestID
				request.Codec = codec
				request.Timeout = int(timeout)
				request.ClassLen = int16(classLen)
				request.HeaderLen = int16(headerLen)
				request.ContentLen = int(contentLen)
				request.ClassName = class
				request.HeaderMap = header
				request.Content = content

				sofarpc.DeserializeBoltRequest(ctx, request)

				cmd = request
				/*
					request := sofarpc.BoltRequestCommand{

						sofarpc.PROTOCOL_CODE_V1,
						dataType,
						int16(cmdCode),
						ver2,
						requestID,
						codec,
						int(timeout),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						nil,
						nil,
					}
					logger.Debugf("BoltV1 DECODE REQUEST, Protocol = %d, CmdType = %d, CmdCode = %d, ReqID = %d",
						request.Protocol, request.CmdType, request.CmdCode, request.ReqID)
					cmd = &request
				*/
			}
		} else if cmdType == sofarpc.RESPONSE {
			//2. response
			if readableBytes >= sofarpc.RESPONSE_HEADER_LEN_V1 {

				cmdCode := binary.BigEndian.Uint16(bytes[2:4])
				ver2 := bytes[4]
				requestID := binary.BigEndian.Uint32(bytes[5:9])
				codec := bytes[9]
				status := binary.BigEndian.Uint16(bytes[10:12])
				classLen := binary.BigEndian.Uint16(bytes[12:14])
				headerLen := binary.BigEndian.Uint16(bytes[14:16])
				contentLen := binary.BigEndian.Uint32(bytes[16:20])

				read = sofarpc.RESPONSE_HEADER_LEN_V1
				var class, header, content []byte

				if readableBytes >= read+int(classLen)+int(headerLen)+int(contentLen) {
					if classLen > 0 {
						class = bytes[read : read+int(classLen)]
						read += int(classLen)
					}
					if headerLen > 0 {
						header = bytes[read : read+int(headerLen)]
						read += int(headerLen)
					}
					if contentLen > 0 {
						content = bytes[read : read+int(contentLen)]
						read += int(contentLen)
					}

					data.Drain(read)
				} else {
					// not enough data
					logger.Debugf("BoltV1 DECODE RESPONSE: no enough data for fully decode")

					return cmd, nil
				}

				//sofabuffers := sofarpc.SofaProtocolBuffersByContext(ctx)
				//response := &sofabuffers.BoltRsp
				response := &sofarpc.BoltResponse{}
				response.Protocol = sofarpc.PROTOCOL_CODE_V1
				response.CmdType = cmdType
				response.CmdCode = int16(cmdCode)
				response.Version = ver2
				response.ReqID = requestID
				response.Codec = codec
				response.ResponseStatus = int16(status)
				response.ClassLen = int16(classLen)
				response.HeaderLen = int16(headerLen)
				response.ContentLen = int(contentLen)
				response.ClassName = class
				response.HeaderMap = header
				response.Content = content

				response.ResponseTimeMillis = time.Now().UnixNano() / int64(time.Millisecond)

				sofarpc.DeserializeBoltResponse(ctx, response)

				cmd = response
				/*
					response := sofarpc.BoltResponseCommand{
						sofarpc.PROTOCOL_CODE_V1,
						dataType,
						int16(cmdCode),
						ver2,
						requestID,
						codec,
						int16(status),
						int16(classLen),
						int16(headerLen),
						int(contentLen),
						class,
						header,
						content,
						nil,
						time.Now().UnixNano() / int64(time.Millisecond),
						nil,
					}

					if cmdCode == uint16(sofarpc.HEARTBEAT) {
						//logger.Debugf("BoltV1 DECODE RESPONSE: Get Bolt HB Msg")
					}
					logger.Debugf("BoltV1 DECODE RESPONSE,ResponseStatus = %d, Protocol = %d, CmdType = %d, CmdCode = %d, ReqID = %d",
						response.ResponseStatus, response.Protocol, response.CmdType, response.CmdCode, response.ReqID)
					cmd = &response
				*/
			}
		} else {
			// 3. unknown type error
			return nil, fmt.Errorf("Decode Error, type = %s, value = %d", sofarpc.UnKnownCmdType, cmdType)
		}
	}

	return cmd, nil
}
