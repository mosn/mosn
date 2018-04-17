package tcpproxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// ReadFilter
type Proxy interface {
	types.ReadFilter

	ReadDisableUpstream(disable bool)

	ReadDisableDownstream(disable bool)
}

type ProxyConfig interface {
	GetRouteFromEntries(connection types.Connection) string
}

type UpstreamCallbacks interface {
	types.ReadFilter
	types.ConnectionEventListener
}

type DownstreamCallbacks interface {
	types.ConnectionEventListener
}

type UpstreamFailureReason string

const (
	ConnectFailed         UpstreamFailureReason = "ConnectFailed"
	NoHealthyUpstream     UpstreamFailureReason = "NoHealthyUpstream"
	ResourceLimitExceeded UpstreamFailureReason = "ResourceLimitExceeded"
	NoRoute               UpstreamFailureReason = "NoRoute"
)
