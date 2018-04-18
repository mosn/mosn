package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/utility"
	"reflect"
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

//PROTOCOL LEVEL's Unified EncodeHeaders for BOLTV1、BOLTV2、TR
func (p *protocols) EncodeHeaders(headers interface{}) (string, types.IoBuffer) {
	var protocolCode byte

	switch headers.(type) {
	case ProtoBasicCmd:
		protocolCode = headers.(ProtoBasicCmd).GetProtocol()
	case map[string]string:
		headers_ := headers.(map[string]string)
		if proto, exist := headers_[SofaPropertyHeader("protocol")]; exist {
			protoValue := ConvertPropertyValue(proto, reflect.Uint8)
			protocolCode = protoValue.(byte)
		} else { //Codec exception
			err := "Invalid encode headers, should contains 'protocol'"
			log.DefaultLogger.Debugf(err)

			//Returned as the exception details
			streamID := utility.GenerateExceptionStreamID(err)
			return streamID, nil
		}
	default:
		err := "Invalid encode headers"
		log.DefaultLogger.Debugf(err)
		streamID := utility.GenerateExceptionStreamID(err)
		return streamID, nil
	}

	log.DefaultLogger.Println("[EncodeHeaders]protocol code = ", protocolCode)

	if proto, exists := p.protocolMaps[protocolCode]; exists {
		//Return encoded data in map[string]string to stream layer
		return proto.GetEncoder().EncodeHeaders(headers)
	} else {
		err := "Unknown protocol code,while encode headers."
		log.DefaultLogger.Debugf("Unknown protocol code: [", protocolCode, "] while encode headers.")
		streamID := utility.GenerateExceptionStreamID(err)
		return streamID, nil
	}
}

func (p *protocols) EncodeData(data types.IoBuffer) types.IoBuffer {
	return data
}

func (p *protocols) EncodeTrailers(trailers map[string]string) types.IoBuffer {
	return nil
}

func (p *protocols) Decode(data types.IoBuffer, filter types.DecodeFilter) {
	// at least 1 byte for protocol code recognize
	for data.Len() > 1 {
		protocolCode := data.Bytes()[0]
		maybeProtocolVersion := data.Bytes()[1]

		log.DefaultLogger.Println("[Decoder]protocol code = ", protocolCode, ", maybeProtocolVersion = ", maybeProtocolVersion)

		if proto, exists := p.protocolMaps[protocolCode]; exists {

			//Decode the Binary Streams to Command Type
			if _, cmd := proto.GetDecoder().Decode(data); cmd != nil {
				proto.GetCommandHandler().HandleCommand(filter, cmd)
			} else {
				break
			}
		} else { //Codec Exception
			headers := make(map[string]string)
			headers[types.HeaderException] = types.MosnExceptionCodeC
			log.DefaultLogger.Println("Unknown protocol code: [", protocolCode, "] while decode in ProtocolDecoder.")

			err := "Unknown protocol code while decode in ProtocolDecoder."
			filter.OnDecodeHeader(utility.GenerateExceptionStreamID(err), headers)
			break
		}
	}
}

func (p *protocols) RegisterProtocol(protocolCode byte, protocol Protocol) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		log.DefaultLogger.Println("Protocol alreay Exist:", protocolCode)
	} else {
		p.protocolMaps[protocolCode] = protocol
	}
}

func (p *protocols) UnRegisterProtocol(protocolCode byte) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		delete(p.protocolMaps, protocolCode)
		log.DefaultLogger.Println("Delete Protocol:", protocolCode)
	}
}

func RegisterProtocol(protocolCode byte, protocol Protocol) {
	defaultProtocols.RegisterProtocol(protocolCode, protocol)
}

func UnRegisterProtocol(protocolCode byte) {
	defaultProtocols.UnRegisterProtocol(protocolCode)
}
