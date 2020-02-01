package tars

import (
	"github.com/TarsCloud/TarsGo/tars"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func init() {
	xprotocol.RegisterMatcher(ProtocolName, tarsMatcher)
}

// predicate dubbo header len and compare magic number
func tarsMatcher(data []byte) types.MatchResult {
	pkgLen, status := tars.TarsRequest(data)
	if pkgLen == 0 && status == tars.PACKAGE_LESS {
		return types.MatchAgain
	}
	if pkgLen == 0 && status == tars.PACKAGE_ERROR {
		return types.MatchFailed
	}
	if status == tars.PACKAGE_FULL {
		return types.MatchSuccess
	}
	return types.MatchFailed
}
