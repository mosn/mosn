package xprotocol

import (
	"context"
	"errors"
	"mosn.io/mosn/pkg/types"
)

type matchPair struct {
	matchFunc types.ProtocolMatch
	protocol  XProtocol
}

// XEngine is an implementation of the ProtocolEngine interface
type XEngine struct {
	protocols []matchPair
}

// Match use registered matchFunc to recognize corresponding protocol
func (engine *XEngine) Match(ctx context.Context, data types.IoBuffer) (types.Protocol, types.MatchResult) {
	again := false

	for idx := range engine.protocols {
		result := engine.protocols[idx].matchFunc(data.Bytes())

		if result == types.MatchSuccess {
			return engine.protocols[idx].protocol, result
		} else if result == types.MatchAgain {
			again = true
		}
	}

	// match not success, return failed if all failed; otherwise return again
	if again {
		return nil, types.MatchAgain
	} else {
		return nil, types.MatchFailed
	}
}

// Register register protocol, which recognized by the matchFunc
func (engine *XEngine) Register(matchFunc types.ProtocolMatch, protocol types.Protocol) error {
	// check name conflict
	for idx := range engine.protocols {
		if engine.protocols[idx].protocol.Name() == protocol.Name() {
			return errors.New("duplicate protocol register:" + string(protocol.Name()))
		}
	}
	xprotocol, ok := protocol.(XProtocol)
	if !ok {
		return errors.New("protocol is not a instance of XProtocol:" + string(protocol.Name()))
	}

	engine.protocols = append(engine.protocols, matchPair{matchFunc: matchFunc, protocol: xprotocol})
	return nil
}

func NewXEngine(protocols []string) (*XEngine, error) {
	engine := &XEngine{}

	for idx := range protocols {
		name := protocols[idx]

		// get protocol
		protocol := GetProtocol(types.ProtocolName(name))
		if protocol == nil {
			return nil, errors.New("no such protocol:" + name)
		}

		// get matcher
		matchFunc := GetMatcher(types.ProtocolName(name))
		if matchFunc == nil {
			return nil, errors.New("protocol matcher is needed while using multiple protocols:" + name)
		}

		// register
		if err := engine.Register(matchFunc, protocol); err != nil {
			return nil, err
		}
	}
	return engine, nil
}
