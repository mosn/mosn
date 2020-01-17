package bolt

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterMatcher(ProtocolName, boltMatcher)
}

// predicate first byte '0x1'
func boltMatcher(data []byte) types.MatchResult {
	length := len(data)
	if length == 0 {
		return types.MatchAgain
	}

	if data[0] == ProtocolCode {
		return types.MatchSuccess
	}

	return types.MatchFailed
}
