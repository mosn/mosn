package original_dst

import (
	"github.com/alipay/sofamosn/pkg/types"
)

type Original_Dst interface {
	OnAccept(cb types.ListenerFilterCallbacks) types.FilterStatus
}
