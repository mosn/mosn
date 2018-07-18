package originaldst

import (
	"github.com/alipay/sofa-mosn/pkg/types"
)

type OriginalDst interface {
	OnAccept(cb types.ListenerFilterCallbacks) types.FilterStatus
}
