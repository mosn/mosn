package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var ClusterAdap ClusterAdapter

type ClusterAdapter struct {
	clusterMng *clusterManager
}

// Called by register module to update cluster's host info
func (ca *ClusterAdapter) TriggerClusterUpdate(clusterName string, hosts []v2.Host) {
	clusterExist := ca.clusterMng.ClusterExist(clusterName)

	if !clusterExist {
		if ca.clusterMng.autoDiscovery {
			cluster := v2.Cluster{
				Name:           clusterName,
				ClusterType:    v2.DYNAMIC_CLUSTER,
				SubClusterType: v2.CONFREG_CLUSTER,
				LbType:         v2.LB_RANDOM,
			}
			
			// for dynamically added cluster, use cluster manager's health check config
			if ca.clusterMng.useHealthCheck {
				cluster.HealthCheck = types.DefaultSofaRpcHealthCheckConf
			}

			ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
		} else {
			log.DefaultLogger.Errorf("doesn't support cluster auto discovery ")
			return
		}
	}

	log.DefaultLogger.Debugf("[TriggerClusterUpdate Called] cluster name is:%s hosts are:%+v",
		clusterName, hosts)
	ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
}

// added when mesh receiving subscribe info
func (ca *ClusterAdapter) TriggerClusterAdded(cluster v2.Cluster) {
	clusterExist := ca.clusterMng.ClusterExist(cluster.Name)

	if !clusterExist {
		log.DefaultLogger.Debugf("Add PrimaryCluster: %s", cluster.Name)

		// for dynamically added cluster, use cluster manager's health check config
		if ca.clusterMng.useHealthCheck {
			cluster.HealthCheck = types.DefaultSofaRpcHealthCheckConf
		}

		ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
	} else {
		log.DefaultLogger.Debugf("Added PrimaryCluster: %s Already Exist", cluster.Name)
	}
}

// called by service unsubscribe
func (ca *ClusterAdapter) TriggerClusterDel(clusterName string) {
	log.DefaultLogger.Debugf("Delete Cluster %s", clusterName)
	ca.clusterMng.RemovePrimaryCluster(clusterName)
}
