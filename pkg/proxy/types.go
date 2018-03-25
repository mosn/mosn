package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"time"
)

type Proxy interface {
	types.ReadFilter

	ReadDisableUpstream(disable bool)

	ReadDisableDownstream(disable bool)
}

type UpstreamCallbacks interface {
	types.ReadFilter
	types.ConnectionCallbacks
}

type DownstreamCallbacks interface {
	types.ConnectionCallbacks
}

type ProxyTimeout struct {
	GlobalTimeout time.Duration
	TryTimeout    time.Duration
}

type UpstreamFailureReason string

const (
	ConnectFailed         UpstreamFailureReason = "ConnectFailed"
	NoHealthyUpstream     UpstreamFailureReason = "NoHealthyUpstream"
	ResourceLimitExceeded UpstreamFailureReason = "ResourceLimitExceeded"
	NoRoute               UpstreamFailureReason = "NoRoute"
)

type UpstreamResetType string

const (
	UpstreamReset         UpstreamResetType = "UpstreamReset"
	UpstreamGlobalTimeout UpstreamResetType = "UpstreamGlobalTimeout"
	UpstreamPerTryTimeout UpstreamResetType = "UpstreamPerTryTimeout"
)