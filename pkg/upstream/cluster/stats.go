package cluster

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func newHostStats(clustername string, addr string) types.HostStats {
	return types.HostStats{
		stats.NewHostStats(clustername, addr),
	}
}

func newClusterStats(clustername string) types.ClusterStats {
	return types.ClusterStats{
		stats.NewClusterStats(clustername),
	}
}
