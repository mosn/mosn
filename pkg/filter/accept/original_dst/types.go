package original_dst

import (
	"github.com/alipay/sofa-mosn/pkg/types"
)

type Original_Dst interface {
	OnAccept(cb types.ListenerFilterCallbacks) types.FilterStatus
}
