package tars

import (
	"context"
	"fmt"

	"github.com/TarsCloud/TarsGo/tars"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterProtocol(ProtocolName, &tarsProtocol{})
}

var MagicTag = []byte{0xda, 0xbb}

type tarsProtocol struct{}

func (proto *tarsProtocol) Name() types.ProtocolName {
	return ProtocolName
}

func (proto *tarsProtocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch model.(type) {
	case Request:
		cmd := model.(*Request)
		return encodeRequest(ctx, cmd)
	case Response:
		cmd := model.(*Response)
		return encodeResponse(ctx, cmd)
	default:
		log.Proxy.Errorf(ctx, "[protocol][tars] encode with unknown command : %+v", model)
	}
	return nil, xprotocol.ErrUnknownType
}

func (proto *tarsProtocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	frameLen, status := tars.TarsRequest(data.Bytes())
	if status == tars.PACKAGE_FULL {
		streamType, err := getStreamType(data.Bytes())
		switch streamType {
		case CmdTypeRequest:
			decodeRequest(ctx, data)
		case CmdTypeResponse:
			decodeResponse(ctx, data)
		default:
			// unknown cmd type
			return nil, fmt.Errorf("[protocol][dubbo] Decode Error, type = %s , err = %v", UnKnownCmdType, err)
		}
		data.Drain(frameLen)
	}
	return nil, nil
}

// heartbeater
func (proto *tarsProtocol) Trigger(requestId uint64) xprotocol.XFrame {
	// not support
	return nil
}

func (proto *tarsProtocol) Reply(requestId uint64) xprotocol.XFrame {
	// not support
	return nil
}

// hijacker
func (proto *tarsProtocol) Hijack(statusCode uint32) xprotocol.XFrame {
	// not support
	return nil
}

func (proto *tarsProtocol) Mapping(httpStatusCode uint32) uint32 {
	// not support
	return -1
}

//判断packet的类型，resonse Packet的包tag=5是字段iRet,int类型；request packet的包tag=5是字段sServantName,string类型
func getStreamType(pkg []byte) (byte, error) {
	// skip pkg length
	pkg = pkg[4:]
	b := codec.NewReader(pkg)
	err, have, ty := b.SkipToNoCheck(5, true)
	if err != nil {
		return CmdTypeUndefine, err
	}
	if !have {
		return CmdTypeUndefine, nil
	}
	switch ty {
	case codec.INT:
		return CmdTypeResponse, nil
	case codec.STRING1:
		return CmdTypeRequest, nil
	case codec.STRING4:
		return CmdTypeRequest, nil
	default:
		return CmdTypeUndefine, nil

	}
	return CmdTypeUndefine, nil
}
