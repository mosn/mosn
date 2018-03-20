package sofarpc


import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type RpcProxy interface {
	types.ReadFilter

	ReadDisableUpstream(disable bool)

	ReadDisableDownstream(disable bool)
	//OTHER Function Definition to FILL
}

type UpstreamCallbacks interface {
	types.ReadFilter
	types.ConnectionCallbacks
}

type DownstreamCallbacks interface {
	types.ConnectionCallbacks
}

type UpstreamFailureReason string

const (
	ConnectFailed         UpstreamFailureReason = "ConnectFailed"
	NoHealthyUpstream     UpstreamFailureReason = "NoHealthyUpstream"
	ResourceLimitExceeded UpstreamFailureReason = "ResourceLimitExceeded"
	NoRoute               UpstreamFailureReason = "NoRoute"
)



