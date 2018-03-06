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

func DefaultProtocols() Protocols {
	return defaultProtocols
}

func NewProtocols(protocolMaps map[byte]Protocol) Protocols {
	return &protocols{
		protocolMaps: protocolMaps,
	}
}

func (p *protocols) Encode(value interface{}, data types.IoBuffer) {}

//TODO move this to seperate type 'ProtocolDecoer' or 'CodecEngine'
func (p *protocols) Decode(ctx interface{}, data types.IoBuffer, out interface{}) {
	readableBytes := uint64(data.Len())

	//at least 1 byte for protocol code recognize
	if readableBytes > 1 {
		bytes := data.Bytes()
		protocolCode := bytes[0]
		maybeProtocolVersion := bytes[1]

		log.DefaultLogger.Println("[Decoder]protocol code = ", protocolCode, ", maybeProtocolVersion = ", maybeProtocolVersion)

		if proto, exists := p.protocolMaps[protocolCode]; exists {
			proto.GetDecoder().Decode(ctx, data, out)
		} else {
			fmt.Println("Unknown protocol code: [", protocolCode, "] while decode in ProtocolDecoder.")
		}
	}
}

//TODO move this to seperate type 'ProtocolDecoer' or 'CodecEngine'
func (p *protocols) Handle(protocolCode byte, ctx interface{}, msg interface{}) {
	if proto, exists := p.protocolMaps[protocolCode]; exists {
		proto.GetCommandHandler().HandleCommand(ctx, msg)
	} else {
		fmt.Println("Unknown protocol code: [", protocolCode, "] while handle in rpc handler.")
	}
}

//put protocol
func (p *protocols) PutProtocol(protocolCode byte, protocol Protocol) {
	p.protocolMaps[protocolCode] = protocol
}

//get protocol
func (p *protocols) GetProtocol(protocolCode byte) Protocol {
	return p.protocolMaps[protocolCode]
}

/**
 * Register protocol with the specified code.
 *
 * @param protocolCode
 * @param protocol
 */
func (p *protocols) RegisterProtocol(protocolCode byte, protocol Protocol) {
	if _, exists := p.protocolMaps[protocolCode]; exists {
		fmt.Println("Protocol alreay Exist:", protocolCode)
	} else {
		p.protocolMaps[protocolCode] = protocol
	}
}

/**
 * Unregister protocol with the specified code.
 *
 * @param protocolCode
 * @return
 */
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
