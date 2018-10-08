package cluster

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func newHostStats(clustername string, addr string) types.HostStats {
	s := stats.NewHostStats(clustername, addr)

	return types.HostStats{
		UpstreamConnectionTotal:                        s.Counter(stats.UpstreamConnectionTotal),
		UpstreamConnectionClose:                        s.Counter(stats.UpstreamConnectionClose),
		UpstreamConnectionActive:                       s.Counter(stats.UpstreamConnectionActive),
		UpstreamConnectionConFail:                      s.Counter(stats.UpstreamConnectionConFail),
		UpstreamConnectionLocalClose:                   s.Counter(stats.UpstreamConnectionLocalClose),
		UpstreamConnectionRemoteClose:                  s.Counter(stats.UpstreamConnectionRemoteClose),
		UpstreamConnectionLocalCloseWithActiveRequest:  s.Counter(stats.UpstreamConnectionLocalCloseWithActiveRequest),
		UpstreamConnectionRemoteCloseWithActiveRequest: s.Counter(stats.UpstreamConnectionRemoteCloseWithActiveRequest),
		UpstreamConnectionCloseNotify:                  s.Counter(stats.UpstreamConnectionCloseNotify),
		UpstreamRequestTotal:                           s.Counter(stats.UpstreamRequestTotal),
		UpstreamRequestActive:                          s.Counter(stats.UpstreamRequestActive),
		UpstreamRequestLocalReset:                      s.Counter(stats.UpstreamRequestLocalReset),
		UpstreamRequestRemoteReset:                     s.Counter(stats.UpstreamRequestRemoteReset),
		UpstreamRequestTimeout:                         s.Counter(stats.UpstreamRequestTimeout),
		UpstreamRequestFailureEject:                    s.Counter(stats.UpstreamRequestFailureEject),
		UpstreamRequestPendingOverflow:                 s.Counter(stats.UpstreamRequestPendingOverflow),
	}
}

func newClusterStats(clustername string) types.ClusterStats {
	s := stats.NewClusterStats(clustername)
	return types.ClusterStats{
		UpstreamConnectionTotal:                        s.Counter(stats.UpstreamConnectionTotal),
		UpstreamConnectionClose:                        s.Counter(stats.UpstreamConnectionClose),
		UpstreamConnectionActive:                       s.Counter(stats.UpstreamConnectionActive),
		UpstreamConnectionConFail:                      s.Counter(stats.UpstreamConnectionConFail),
		UpstreamConnectionRetry:                        s.Counter(stats.UpstreamConnectionRetry),
		UpstreamConnectionLocalClose:                   s.Counter(stats.UpstreamConnectionLocalClose),
		UpstreamConnectionRemoteClose:                  s.Counter(stats.UpstreamConnectionRemoteClose),
		UpstreamConnectionLocalCloseWithActiveRequest:  s.Counter(stats.UpstreamConnectionLocalCloseWithActiveRequest),
		UpstreamConnectionRemoteCloseWithActiveRequest: s.Counter(stats.UpstreamConnectionRemoteCloseWithActiveRequest),
		UpstreamConnectionCloseNotify:                  s.Counter(stats.UpstreamConnectionCloseNotify),
		UpstreamBytesReadTotal:                         s.Counter(stats.UpstreamBytesReadTotal),
		UpstreamBytesWriteTotal:                        s.Counter(stats.UpstreamBytesWriteTotal),
		UpstreamRequestTotal:                           s.Counter(stats.UpstreamRequestTotal),
		UpstreamRequestActive:                          s.Counter(stats.UpstreamRequestActive),
		UpstreamRequestLocalReset:                      s.Counter(stats.UpstreamRequestLocalReset),
		UpstreamRequestRemoteReset:                     s.Counter(stats.UpstreamRequestRemoteReset),
		UpstreamRequestRetry:                           s.Counter(stats.UpstreamRequestRetry),
		UpstreamRequestRetryOverflow:                   s.Counter(stats.UpstreamRequestRetryOverflow),
		UpstreamRequestTimeout:                         s.Counter(stats.UpstreamRequestTimeout),
		UpstreamRequestFailureEject:                    s.Counter(stats.UpstreamRequestFailureEject),
		UpstreamRequestPendingOverflow:                 s.Counter(stats.UpstreamRequestPendingOverflow),
		LBSubSetsFallBack:                              s.Counter(stats.UpstreamLBSubSetsFallBack),
		LBSubSetsActive:                                s.Counter(stats.UpstreamLBSubSetsActive),
		LBSubsetsCreated:                               s.Counter(stats.UpstreamLBSubsetsCreated),
		LBSubsetsRemoved:                               s.Counter(stats.UpstreamLBSubsetsRemoved),
	}
}
