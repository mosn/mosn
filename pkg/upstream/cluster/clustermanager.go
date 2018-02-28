package cluster

import (
	"net"
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

const (
	ClusterSnapshot = "ClusterSnapshot"
)

// ClusterManager
type clusterManager struct {
	sourceAddr       net.Addr
	primaryClusters  map[string]*primaryCluster
	clusterSnapshots map[string]*golocalstore
}

type clusterSnapshot struct {
	prioritySet  types.PrioritySet
	clusterInfo  types.ClusterInfo
	loadbalancer types.LoadBalancer
}

func NewClusterManager(sourceAddr net.Addr) types.ClusterManager {
	return &clusterManager{
		sourceAddr:       sourceAddr,
		primaryClusters:  make(map[string]*primaryCluster),
		clusterSnapshots: make(map[string]*golocalstore),
	}
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
	found := false
	clusterName := cluster.Name

	for _, pc := range cm.primaryClusters {
		if pc.cluster.Info().Name() == clusterName {
			if !pc.addedViaApi {
				// cant update status-config cluster
				return false
			} else {
				found = true
				break
			}
		}
	}

	if found {
		delete(cm.primaryClusters, clusterName)
	}

	cm.loadCluster(cluster, true)

	return true
}

func (cm *clusterManager) loadCluster(clusterConfig v2.Cluster, addedViaApi bool) types.Cluster {
	cluster := NewCluster(clusterConfig, cm.sourceAddr, addedViaApi)

	cluster.Initialize(func() {
		cluster.PrioritySet().AddMemberUpdateCb(func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
			cm.updateClusterSnapshot(cluster, priority, hostsAdded, hostsRemoved)
		})
	})

	cm.primaryClusters[clusterConfig.Name] = &primaryCluster{
		cluster:     cluster,
		addedViaApi: addedViaApi,
	}

	return cluster
}

func (cm *clusterManager) getOrCreateClusterSnapshot(clusterName string) *clusterSnapshot {
	var ok bool
	var snapshotStore *golocalstore
	var primaryCluster *primaryCluster

	if snapshotStore, ok = cm.clusterSnapshots[clusterName]; ok {
		if _, ok := cm.primaryClusters[clusterName]; !ok {
			return nil
		} else {
			cm.clusterSnapshots[clusterName] = newgolocalstore()
		}
	}

	snapshot := snapshotStore.Get(ClusterSnapshot)

	if snapshot != nil {
		// TODO: use prioritySet copy and clusterInfo copy
		ps := primaryCluster.cluster.PrioritySet()
		ci := primaryCluster.cluster.Info()

		clusterSnapshot := &clusterSnapshot{
			prioritySet:  ps,
			clusterInfo:  ci,
			loadbalancer: NewLoadBalancer(primaryCluster.cluster.Info().LbType(), ps),
		}

		snapshot = clusterSnapshot
		snapshotStore.Set(ClusterSnapshot, clusterSnapshot)
	}

	return snapshot.(*clusterSnapshot)
}

func (cm *clusterManager) updateClusterSnapshot(cluster types.Cluster, priority uint32,
	hostsAdded []types.Host, hostsRemoved []types.Host) {
	hostSet := cluster.PrioritySet().HostSetsByPriority()[priority]

	var hosts []types.Host
	copy(hosts, hostSet.Hosts())

	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster.Info().Name())
	clusterSnapshot.prioritySet.GetOrCreateHostSet(priority).UpdateHosts(hosts, nil,
		nil, nil, hostsAdded, hostsRemoved)

	cm.clusterSnapshots[cluster.Info().Name()].Set(ClusterSnapshot, clusterSnapshot)
}

func (cm *clusterManager) SetInitializedCb(cb func()) {}

func (cm *clusterManager) Clusters() map[string]types.Cluster {
	clusterInfoMap := make(map[string]types.Cluster)

	for c, pc := range cm.primaryClusters {
		clusterInfoMap[c] = pc.cluster
	}

	return clusterInfoMap
}

func (cm *clusterManager) Get(cluster string, context context.Context) types.ClusterSnapshot {
	return cm.getOrCreateClusterSnapshot(cluster)
}

func (cm *clusterManager) HttpConnPoolForCluster(cluster string, priority pkg.Priority, protocol string, context context.Context) types.HttpConnectionPool {
	return nil
}

func (cm *clusterManager) TcpConnForCluster(cluster string, context context.Context) types.CreateConnectionData {
	clusterSnapshot := cm.getOrCreateClusterSnapshot(cluster)

	if clusterSnapshot == nil {
		return types.CreateConnectionData{}
	}

	host := clusterSnapshot.loadbalancer.ChooseHost(nil)

	if host != nil {
		return host.CreateConnection()
	} else {
		return types.CreateConnectionData{}
	}
}

func (cm *clusterManager) RemovePrimaryCluster(cluster string) {

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
