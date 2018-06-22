package protocol

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

const (
	SofaRpc types.Protocol = "SofaRpc"
	Http1   types.Protocol = "Http1"
	Http2   types.Protocol = "Http2"
)

const (
	MosnHeadersHostKey = "HostKey"
	MosnHeaderPathKey = "PathKey"
	MosnHeaderQueryStringKey = "QueryStringKey"
)