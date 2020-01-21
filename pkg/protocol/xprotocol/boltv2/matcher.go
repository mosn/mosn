package boltv2

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterMatcher(ProtocolName, boltv2Matcher)
}

// predicate first byte '0x2'
func boltv2Matcher(data []byte) types.MatchResult {
	length := len(data)
	if length == 0 {
		return types.MatchAgain
	}

	if data[0] == ProtocolCode {
		return types.MatchSuccess
	}

	return types.MatchFailed
}
