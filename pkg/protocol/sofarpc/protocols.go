package sofarpc

import (
	"context"
	"reflect"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//All of the protocolMaps

var defaultProtocols = &protocols{
	protocolMaps: make(map[byte]Protocol),
}

type protocols struct {
	protocolMaps map[byte]Protocol
}

func DefaultProtocols() types.Protocols {
	return defaultProtocols
}

func NewProtocols(protocolMaps map[byte]Protocol) types.Protocols {
	return &protocols{
		protocolMaps: protocolMaps,
	}
}

// todo: add error as return value
//PROTOCOL LEVEL's Unified EncodeHeaders for BOLTV1、BOLTV2、TR
func (p *protocols) EncodeHeaders(context context.Context, headers interface{}) (string, types.IoBuffer) {
	var protocolCode byte

	switch headers.(type) {
	case ProtoBasicCmd:
		protocolCode = headers.(ProtoBasicCmd).GetProtocol()
	case map[string]string:
		headersMap := headers.(map[string]string)

		if proto, exist := headersMap[SofaPropertyHeader(HeaderProtocolCode)]; exist {
			protoValue := ConvertPropertyValue(proto, reflect.Uint8)
			protocolCode = protoValue.(byte)
		} else {
			//Codec exception
			log.ByContext(context).Errorf("Invalid encode headers, should contains 'protocol'")

			return "", nil
		}
	default:
		err := "Invalid encode headers"
		log.ByContext(context).Errorf(err)

		return "", nil
	}
	//todo: for @ledou to fix the contex bug
	//log.ByContext(context).Debugf("[EncodeHeaders]protocol code = %x", protocolCode)

	if proto, exists := p.protocolMaps[protocolCode]; exists {
		//Return encoded data in map[string]string to stream layer
		return proto.GetEncoder().EncodeHeaders(context, headers)
	} else {
		log.ByContext(context).Errorf("Unknown protocol code: [%d] while encode headers.", protocolCode)

		return "", nil
	}
}

func (p *protocols) EncodeData(context context.Context, data types.IoBuffer) types.IoBuffer {
	return data
}

func (p *protocols) EncodeTrailers(context context.Context, trailers map[string]string) types.IoBuffer {
	return nil
}

func (p *protocols) Decode(context context.Context, data types.IoBuffer, filter types.DecodeFilter) {
	// at least 1 byte for protocol code recognize
	for data.Len() > 1 {
		logger := log.ByContext(context)

		protocolCode := data.Bytes()[0]
		maybeProtocolVersion := data.Bytes()[1]

		logger.Debugf("Decoderprotocol code = %x, maybeProtocolVersion = %x", protocolCode, maybeProtocolVersion)

		if proto, exists := p.protocolMaps[protocolCode]; exists {

			//Decode the Binary Streams to Command Type
			if _, cmd := proto.GetDecoder().Decode(context, data); cmd != nil {
				proto.GetCommandHandler().HandleCommand(context, cmd, filter)
			} else {
				break
			}
		} else {
			//Codec Exception
			headers := make(map[string]string, 1)
			headers[types.HeaderException] = types.MosnExceptionCodeC
			logger.Errorf("Unknown protocol code: [%x] while decode in ProtocolDecoder.", protocolCode)

			err := "Unknown protocol code while decode in ProtocolDecoder."
			filter.OnDecodeHeader(GenerateExceptionStreamID(err), headers)

			break
		}
	}
}

func (p *protocols) RegisterProtocol(protocolCode byte, protocol Protocol) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		log.DefaultLogger.Warnf("protocol alreay Exist:", protocolCode)
	} else {
		p.protocolMaps[protocolCode] = protocol
		log.StartLogger.Debugf("register protocol:%x", protocolCode)
	}
}

func (p *protocols) UnRegisterProtocol(protocolCode byte) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		delete(p.protocolMaps, protocolCode)
		log.StartLogger.Debugf("unregister protocol:%x", protocolCode)
	}
}

func RegisterProtocol(protocolCode byte, protocol Protocol) {
	defaultProtocols.RegisterProtocol(protocolCode, protocol)
}

func UnRegisterProtocol(protocolCode byte) {
	defaultProtocols.UnRegisterProtocol(protocolCode)
}
