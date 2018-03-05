package codec

import (
	"bytes"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

//All of the protocols

type Protocols struct {
	protocols map[byte]protocol.Protocol
}

func NewProtocols(protocols map[byte]protocol.Protocol) Protocols{
	return Protocols{protocols}
}

func (p *Protocols) Encode(value interface{}, data bytes.Buffer) {

}

//TODO move this to seperate type 'ProtocolDecoer' or 'CodecEngine'
func (p *Protocols) Decode(ctx interface{}, data *bytes.Buffer, out interface{}) {
	readableBytes := uint64(data.Len())
	//at least 1 byte for protocol code recognize
	if readableBytes > 1 {

		//TODO mark pos and reset, otherwise protocl.decoder can only get the rest of the data without the 2 bytes at head
		protocolCode, err := data.ReadByte()
		if err != nil {
			fmt.Println("decode protocol code error :", err)
		}
		/**
		maybeProtocolVersion, err := data.ReadByte()
		if err != nil {
			fmt.Println("decode protocol version error :", err)
		} else {
			fmt.Println("2nd byte:", maybeProtocolVersion)
		}**/

		if proto, exists := p.protocols[protocolCode]; exists {
			proto.GetDecoder().Decode(ctx, data, out)
		} else {
			fmt.Println("Unknown protocol code: [", protocolCode, "] while decode in ProtocolDecoder.")
		}
	}
}

//TODO move this to seperate type 'ProtocolDecoer' or 'CodecEngine'
func (p *Protocols) Handle(protocolCode byte, ctx interface{}, msg interface{}) {
		if proto, exists := p.protocols[protocolCode]; exists {
			proto.GetCommandHandler().HandleCommand(ctx, msg)
		} else {
			fmt.Println("Unknown protocol code: [", protocolCode, "] while handle in rpc handler.")
		}
}

//put protocol
func (p *Protocols) PutProtocol(protocolCode byte, protocol protocol.Protocol) {
	p.protocols[protocolCode] = protocol
}

//get protocol
func (p *Protocols) GetProtocol(protocolCode byte) protocol.Protocol {
	return p.protocols[protocolCode]
}

/**
 * Register protocol with the specified code.
 *
 * @param protocolCode
 * @param protocol
 */
func (p *Protocols) RegisterProtocol(protocolCode byte, protocol protocol.Protocol) {
	if _, exists := p.protocols[protocolCode]; exists {
		fmt.Println("Protocol alreay Exist:", protocolCode)
	} else {
		p.protocols[protocolCode] = protocol
	}
}

/**
 * Unregister protocol with the specified code.
 *
 * @param protocolCode
 * @return
 */
func (p *Protocols) UnRegisterProtocol(protocolCode byte) {
	if _, exists := p.protocols[protocolCode]; exists {
		delete(p.protocols, protocolCode)
		fmt.Println("Delete Protocol:", protocolCode)
	}
}
