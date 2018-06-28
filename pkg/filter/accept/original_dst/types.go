package original_dst

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type Original_Dst interface {
	OnAccept(cb types.ListenerFilterCallbacks) types.FilterStatus
}