package xprotocol

import (
	"sofastack.io/sofa-mosn/pkg/types"
	"errors"
)

var (
	protocolMap = make(map[types.ProtocolName]XProtocol)
	matcherMap  = make(map[types.ProtocolName]types.ProtocolMatch)
)

func RegisterProtocol(name types.ProtocolName, protocol XProtocol) error {
	// check name conflict
	_, ok := protocolMap[name]
	if ok {
		return errors.New("duplicate protocol register:" + string(name))
	}

	protocolMap[name] = protocol
	return nil
}

func GetProtocol(name types.ProtocolName) XProtocol {
	return protocolMap[name]
}

func RegisterMatcher(name types.ProtocolName, matcher types.ProtocolMatch) error {
	// check name conflict
	_, ok := matcherMap[name]
	if ok {
		return errors.New("duplicate matcher register:" + string(name))
	}

	matcherMap[name] = matcher
	return nil
}

func GetMatcher(name types.ProtocolName) types.ProtocolMatch {
	return matcherMap[name]
}
