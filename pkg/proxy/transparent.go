package proxy

import (
	"net"

	v2 "mosn.io/mosn/pkg/config/v2"
	clusterAdapter "mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/types"
)

func genTransparentRoute() types.Route {

	rr := router.NewTransparentRouteImpl()

	return rr
}

func createTransparentCluster() types.Cluster {
	clusterConfig := v2.Cluster{
		Name:   "transparent_cluster",
		LbType: v2.LB_RANDOM,
		LBSubSetConfig: v2.LBSubsetConfig{
			FallBackPolicy: 1,
			SubsetSelectors: [][]string{
				[]string{"version"},
				[]string{"version", "zone"},
			},
		},
	}
	return clusterAdapter.NewCluster(clusterConfig)
}

func getTransparentRoute(addr net.Addr) (types.ClusterSnapshot, types.Route) {
	cluster := createTransparentCluster()
	snap := cluster.Snapshot()

	config := v2.Host{
		HostConfig: v2.HostConfig{
			Address:    addr.String(),
		},
	}

	host := clusterAdapter.NewSimpleHost(config, snap.ClusterInfo())
	cluster.UpdateHosts([]types.Host{host})

	snap = cluster.Snapshot()

	r := router.NewTransparentRouteImpl()

	return snap, r
}
