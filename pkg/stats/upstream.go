package stats

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
)

// upstream metrics type
const UpstreamType = "upstream"

//  key in cluster/host
const (
	UpstreamConnectionTotal                        = "upstream_connection_total"
	UpstreamConnectionClose                        = "upstream_connection_close"
	UpstreamConnectionActive                       = "upstream_connection_active"
	UpstreamConnectionConFail                      = "upstream_connection_con_fail"
	UpstreamConnectionRetry                        = "upstream_connection_retry"
	UpstreamConnectionLocalClose                   = "upstream_connection_local_close"
	UpstreamConnectionRemoteClose                  = "upstream_connection_remote_close"
	UpstreamConnectionLocalCloseWithActiveRequest  = "upstream_connection_local_close_with_active_request"
	UpstreamConnectionRemoteCloseWithActiveRequest = "upstream_connection_remote_close_with_active_request"
	UpstreamConnectionCloseNotify                  = "upstream_connection_close_notify"
	UpstreamRequestTotal                           = "upstream_request_request_total"
	UpstreamRequestActive                          = "upstream_request_request_active"
	UpstreamRequestLocalReset                      = "upstream_request_request_local_reset"
	UpstreamRequestRemoteReset                     = "upstream_request_request_remote_reset"
	UpstreamRequestTimeout                         = "upstream_request_request_timeout"
	UpstreamRequestFailureEject                    = "upstream_request_failure_eject"
	UpstreamRequestPendingOverflow                 = "upstream_request_pending_overflow"
)

//  key in cluster
const (
	UpstreamRequestRetry         = "upstream_request_retry"
	UpstreamRequestRetryOverflow = "upstream_request_retry_overfolw"
	UpstreamLBSubSetsFallBack    = "upstream_lb_subsets_fallback"
	UpstreamLBSubSetsActive      = "upstream_lb_subsets_active"
	UpstreamLBSubsetsCreated     = "upstream_lb_subsets_created"
	UpstreamLBSubsetsRemoved     = "upstream_lb_subsets_removed"
	UpstreamBytesRead            = "upstream_connection_bytes_read"
	UpstreamBytesReadCurrent     = "upstream_connection_bytes_read_current"
	UpstreamBytesWrite           = "upstream_connection_bytes_write"
	UpstreamBytesWriteCurrent    = "upstream_connection_bytes_write_current"
)

func NewHostStats(clustername string, addr string) types.Metrics {
	namespace := fmt.Sprintf("cluster.%s.host.%s", clustername, addr)
	return NewStats(UpstreamType, namespace)
}

func NewClusterStats(clustername string) types.Metrics {
	namespace := fmt.Sprintf("cluster.%s", clustername)
	return NewStats(UpstreamType, namespace)
}
