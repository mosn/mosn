package stats

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// downstream metrics type
const DownstreamType = "downstream"

// metrics key in listener/proxy
const (
	DownstreamConnectionTotal   = "downstream_connection_total"
	DownstreamConnectionDestroy = "downstream_connection_destroy"
	DownstreamConnectionActive  = "downstream_connection_active"
	DownstreamBytesRead         = "downstream_bytes_read"
	DownstreamBytesReadCurrent  = "downstream_bytes_read_current"
	DownstreamBytesWrite        = "downstream_bytes_write"
	DownstreamBytesWriteCurrent = "downstream_bytes_write_current"
	DownstreamRequestTotal      = "downstream_request_total"
	DownstreamRequestActive     = "downstream_request_active"
	DownstreamRequestReset      = "downstream_request_reset"
	DownstreamRequestTime       = "downstream_request_time"
)

func NewProxyStats(proxyname string) types.Metrics {
	namespace := fmt.Sprintf("proxy_%s", proxyname)
	return NewStats(DownstreamType, namespace)
}

func NewListenerStats(listenername string) types.Metrics {
	namespace := fmt.Sprintf("listener_%s", listenername)
	return NewStats(DownstreamType, namespace)
}
