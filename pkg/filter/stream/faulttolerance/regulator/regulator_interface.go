package regulator

import (
	"mosn.io/mosn/pkg/filter/stream/faulttolerance/invocation"
)

type Regulator interface {
	Regulate(stat *invocation.InvocationStat)
}
