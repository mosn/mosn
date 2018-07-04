package cluster

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream/http2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream/xprotocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// ClusterManager
type clusterManager struct {
	sourceAddr        net.Addr
	primaryClusters   cmap.ConcurrentMap // string: *primaryCluster
	sofaRpcConnPool   cmap.ConcurrentMap // string: types.ConnectionPool
	http2ConnPool     cmap.ConcurrentMap // string: types.ConnectionPool
	xProtocolConnPool cmap.ConcurrentMap // string: types.ConnectionPool
	clusterAdapter    ClusterAdapter
	autoDiscovery     bool
}

type clusterSnapshot struct {
	prioritySet  types.PrioritySet
	clusterInfo  types.ClusterInfo
	loadbalancer types.LoadBalancer
}

func NewClusterManager(sourceAddr net.Addr, clusters []v2.Cluster,
	clusterMap map[string][]v2.Host, autoDiscovery bool) types.ClusterManager {
	cm := &clusterManager{
		sourceAddr:        sourceAddr,
		primaryClusters:   cmap.New(),
		sofaRpcConnPool:   cmap.New(),
		http2ConnPool:     cmap.New(),
		xProtocolConnPool: cmap.New(),
		autoDiscovery:     autoDiscovery,
	}
	//init ClusterAdap when run app
	ClusterAdap = ClusterAdapter{
		clusterMng: cm,
	}

	cm.clusterAdapter = ClusterAdap

	//Add cluster to cm
	//Register upstream update type
	for _, cluster := range clusters {
		cm.AddOrUpdatePrimaryCluster(cluster)
	}

	//Add hosts to cluster
	for clusterName, hosts := range clusterMap {
		cm.UpdateClusterHosts(clusterName, 0, hosts)
	}

	return cm
}

func (cs *clusterSnapshot) PrioritySet() types.PrioritySet {
	return cs.prioritySet
}

func (cs *clusterSnapshot) ClusterInfo() types.ClusterInfo {
	return cs.clusterInfo
}

func (cs *clusterSnapshot) LoadBalancer() types.LoadBalancer {
	return cs.loadbalancer
}

type primaryCluster struct {
	cluster     types.Cluster
	addedViaApi bool
}

func (cm *clusterManager) AddOrUpdatePrimaryCluster(cluster v2.Cluster) bool {
	clusterName := cluster.Name

	if v, exist := cm.primaryClusters.Get(clusterName); exist {
		if !v.(*primaryCluster).addedViaApi {
			return false
		}
		//return true
	}

	cm.loadCluster(cluster, true)

	return true
}

func (cm *clusterManager) ClusterExist(clusterName string) bool {
	if _, exist := cm.primaryClusters.Get(clusterName); exist {
		return true
	} else {
		return false
	}
}

func (cm *clusterManager) loadCluster(clusterConfig v2.Cluster, addedViaApi bool) types.Cluster {
	cluster := NewCluster(clusterConfig, cm.sourceAddr, addedViaApi)

	cluster.Initialize(func() {
		cluster.PrioritySet().AddMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
		})
	})

	cm.primaryClusters.Set(clusterConfig.Name, &primaryCluster{
		cluster:     cluster,
		addedViaApi: addedViaApi,
	})

	return cluster
}

func (cm *clusterManager) getOrCreateClusterSnapshot(clusterName string) *clusterSnapshot {
	if v, ok := cm.primaryClusters.Get(clusterName); ok {
		pcc := v.(*primaryCluster).cluster

		clusterSnapshot := &clusterSnapshot{
			prioritySet:  pcc.PrioritySet(),
			clusterInfo:  pcc.Info(),
			loadbalancer: NewLoadBalancer(pcc.Info().LbType(), pcc.PrioritySet()),
		}

		return clusterSnapshot
	} else {
		return nil
	}
}

func (cm *clusterManager) SetInitializedCb(cb func()) {}

func (cm *clusterManager) Clusters() map[string]types.Cluster {
	clusterInfoMap := make(map[string]types.Cluster)

	for c, pc := range cm.primaryClusters.Items() {
		clusterInfoMap[c] = pc.(*primaryCluster).cluster
	}

	return clusterInfoMap
}

func (cm *clusterManager) Get(cluster string, context context.Context) types.ClusterSnapshot {
	return cm.getOrCreateClusterSnapshot(cluster)
}

func (cm *clusterManager) UpdateClusterHosts(clusterName string, priority uint32, hostConfigs []v2.Host) error {
	if v, ok := cm.primaryClusters.Get(clusterName); ok {
		pcc := v.(*primaryCluster).cluster

		// todo: hack
		if concretedCluster, ok := pcc.(*simpleInMemCluster); ok {
			var hosts []types.Host

			for _, hc := range hostConfigs {
				hosts = append(hosts, NewHost(hc, pcc.Info()))
			}
			concretedCluster.UpdateHosts(hosts)
			return nil
		} else {
			return errors.New(fmt.Sprintf("cluster's hostset %s can't be update", clusterName))
		}
	}

	return errors.New(fmt.Sprintf("cluster %s not found", clusterName))
}

func (cm *clusterManager) HttpConnPoolForCluster(cluster string, protocol types.Protocol,
	context context.Context) types.ConnectionPool {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		return nil
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(nil)

	if host != nil {
		addr := host.AddressString()
		log.StartLogger.Tracef("http connection pool upstream addr : %v", addr)

		// todo: support protocol http1.x
		if connPool, ok := cm.http2ConnPool.Get(addr); ok {
			return connPool.(types.ConnectionPool)
		} else {
			// todo: move this to a centralized factory, remove dependency to http2 stream
			connPool := http2.NewConnPool(host)
			cm.http2ConnPool.Set(addr, connPool)

			return connPool
		}
	} else {
		return nil
	}
}

func (cm *clusterManager) XprotocolConnPoolForCluster(cluster string, protocol types.Protocol,
	context context.Context) types.ConnectionPool {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		return nil
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(nil)

	if host != nil {
		addr := host.AddressString()
		log.StartLogger.Tracef("Xprotocol connection pool upstream addr : %v", addr)

		if connPool, ok := cm.xProtocolConnPool.Get(addr); ok {
			return connPool.(types.ConnectionPool)
		} else {
			connPool := xprotocol.NewConnPool(host)
			cm.xProtocolConnPool.Set(addr, connPool)

			return connPool
		}
	} else {
		return nil
	}
}

func (cm *clusterManager) TcpConnForCluster(cluster string, context context.Context) types.CreateConnectionData {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		return types.CreateConnectionData{}
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(nil)

	if host != nil {
		return host.CreateConnection(context)
	} else {
		return types.CreateConnectionData{}
	}
}

func (cm *clusterManager) SofaRpcConnPoolForCluster(cluster string, context context.Context) types.ConnectionPool {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		log.DefaultLogger.Errorf(" Sofa Rpc ConnPool For Cluster is nil %s", cluster)
		return nil
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(nil)

	if host != nil {
		addr := host.AddressString()
		log.DefaultLogger.Debugf(" clusterSnapshot.loadbalancer.ChooseHost result is %s,cluser is %s", addr, cluster)

		if connPool, ok := cm.sofaRpcConnPool.Get(addr); ok {
			return connPool.(types.ConnectionPool)
		} else {
			// todo: move this to a centralized factory, remove dependency to sofarpc stream
			connPool := sofarpc.NewConnPool(host)
			cm.sofaRpcConnPool.Set(addr, connPool)

			return connPool
		}
	} else {
		log.DefaultLogger.Errorf("  clusterSnapshot.loadbalancer.ChooseHost is nil %s", cluster)
		return nil
	}
}

func (cm *clusterManager) RemovePrimaryCluster(clusterName string) bool {
	if v, exist := cm.primaryClusters.Get(clusterName); exist {
		if !v.(*primaryCluster).addedViaApi {
			return false
		}
	}

	cm.primaryClusters.Remove(clusterName)

	return true
}

func (cm *clusterManager) Shutdown() error {
	return nil
}

func (cm *clusterManager) SourceAddress() net.Addr {
	return cm.sourceAddr
}

func (cm *clusterManager) VersionInfo() string {
	return ""
}

func (cm *clusterManager) LocalClusterName() string {
	return ""
}
