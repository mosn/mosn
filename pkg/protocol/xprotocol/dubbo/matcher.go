package dubbo

import (
	"bytes"

	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterMatcher(ProtocolName, dubboMatcher)
}

// predicate dubbo header len and compare magic number
func dubboMatcher(data []byte) types.MatchResult {
	if len(data) < HeaderLen {
		return types.MatchAgain
	}
	if bytes.Compare(data[MagicIdx:FlagIdx], MagicTag) != 0 {
		return types.MatchFailed
	}
	return types.MatchSuccess
}
