package sofarpc

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
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

func (p *protocols) EncodeHeaders(headers map[string]string) (uint32, types.IoBuffer) {
	protocolCode := []byte(headers[SofaPropertyHeader("protocol")])[0]
	log.DefaultLogger.Println("[EncodeHeaders]protocol code = ", protocolCode)

	if proto, exists := p.protocolMaps[protocolCode]; exists {
		return proto.GetEncoder().EncodeHeaders(headers) //返回ENCODE的数据
	} else {
		log.DefaultLogger.Println("Unknown protocol code: [", protocolCode, "] while encode in ProtocolDecoder.")

		return 0, nil
	}
}

func (p *protocols) EncodeData(data types.IoBuffer) types.IoBuffer {
	return data
}

func (p *protocols) EncodeTrailers(trailers map[string]string) types.IoBuffer {
	return nil
}

// filter = type.serverStreamConnection
func (p *protocols) Decode(data types.IoBuffer, filter types.DecodeFilter) {
	//at least 1 byte for protocol code recognize
	for data.Len() > 1 {
		protocolCode := data.Bytes()[0]
		maybeProtocolVersion := data.Bytes()[1]

		log.DefaultLogger.Println("[Decoder]protocol code = ", protocolCode, ", maybeProtocolVersion = ", maybeProtocolVersion)

		if proto, exists := p.protocolMaps[protocolCode]; exists {
			decoder := proto.GetDecoder()
			_, cmd := decoder.Decode(data) //先解析成command,即将一串二进制Decode到对应的字段

			if cmd != nil {
				proto.GetCommandHandler().HandleCommand(filter, cmd) //做decode 同时序列化，在此调用！！
			} else {
				log.DefaultLogger.Debugf("Unable to decode sofa rpc command")
				break
			}
		} else {
			log.DefaultLogger.Debugf("Unknown protocol code: [", protocolCode, "] while decode in ProtocolDecoder.")
			break
		}
	}
}

func (p *protocols) RegisterProtocol(protocolCode byte, protocol Protocol) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		fmt.Println("Protocol alreay Exist:", protocolCode)
	} else {
		p.protocolMaps[protocolCode] = protocol
	}
}

func (p *protocols) UnRegisterProtocol(protocolCode byte) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		delete(p.protocolMaps, protocolCode)
		fmt.Println("Delete Protocol:", protocolCode)
	}
}

func RegisterProtocol(protocolCode byte, protocol Protocol) {
	defaultProtocols.RegisterProtocol(protocolCode, protocol)
}

func UnRegisterProtocol(protocolCode byte) {
	defaultProtocols.UnRegisterProtocol(protocolCode)
}
