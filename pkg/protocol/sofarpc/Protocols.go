package sofarpc

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"fmt"
)

//All of the protocols

type Protocols struct{

	protocols    		map[byte]protocol.Protocol
	ProtocolsBolt		map[byte]bolt



}



func (p*Protocols)PutProtocolBolt(protocol_code byte,protocol bolt){

	p.ProtocolsBolt[protocol_code] = protocol
}

//get protocol
func (p*Protocols)GetProtocolBolt(protocol_code byte) bolt{

	return p.ProtocolsBolt[protocol_code]
}



//put protocol
func (p*Protocols)PutProtocol(protocol_code byte,protocol protocol.Protocol){

	p.protocols[protocol_code] = protocol
}

//get protocol
func (p*Protocols)GetProtocol(protocol_code byte) protocol.Protocol{

	return p.protocols[protocol_code]
}


/**
 * Register protocol with the specified code.
 *
 * @param protocolCode
 * @param protocol
 */

func (p*Protocols)RegisterProtocol(protocol_code byte,protocol protocol.Protocol){

	_,exists := p.protocols[protocol_code]
	if exists {
		fmt.Println("Protocol alreay Exist:",protocol_code)

	}else{

		p.protocols[protocol_code] = protocol
	}

}

/**
 * Unregister protocol with the specified code.
 *
 * @param protocolCode
 * @return
 */

 func (p*Protocols)UnRegisterProtocol(protocol_code byte){

	 _,exists := p.protocols[protocol_code]
	 if exists {
	 	delete(p.protocols,protocol_code)
	 	fmt.Println("Delete Protocol:",protocol_code)
	 }
 }









