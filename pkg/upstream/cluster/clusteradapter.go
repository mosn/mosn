package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"errors"
)

var ClusterAdap ClusterAdapter

type ClusterAdapter struct {
	clusterMng *clusterManager
}

// Called by registry module to update cluster's host info
func (ca *ClusterAdapter) TriggerClusterUpdate(clusterName string, hosts []v2.Host) error {
	clusterExist := ca.clusterMng.ClusterExist(clusterName)

	if !clusterExist {
		if ca.clusterMng.autoDiscovery {
			cluster := v2.Cluster{
				Name:           clusterName,
				ClusterType:    v2.DYNAMIC_CLUSTER,
				LbType:         v2.LB_RANDOM,
			}
			
			// for dynamically added cluster, use cluster manager's health check config
			if ca.clusterMng.registryUseHealthCheck {
				// todo support more default health check @boqin
				cluster.HealthCheck = sofarpc.DefaultSofaRpcHealthCheckConf
			}

			ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
		} else {
			msg := "cluster doesn't support auto discovery "
			log.DefaultLogger.Errorf(msg)
			return errors.New(msg)
		}
	}

	log.DefaultLogger.Debugf("triggering cluster update, cluster name = %s hosts = %+v",clusterName, hosts)
	ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
	
	return nil
}

// Called when mesh receive subscribe info
func (ca *ClusterAdapter) TriggerClusterAdded(cluster v2.Cluster) {
	clusterExist := ca.clusterMng.ClusterExist(cluster.Name)

	if !clusterExist {
		log.DefaultLogger.Debugf("Add PrimaryCluster: %s", cluster.Name)

		// for dynamically added cluster, use cluster manager's health check config
		if ca.clusterMng.registryUseHealthCheck {
			cluster.HealthCheck = sofarpc.DefaultSofaRpcHealthCheckConf
		}

		ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
	} else {
		log.DefaultLogger.Debugf("Added PrimaryCluster: %s Already Exist", cluster.Name)
	}
}

// Called when mesh receive unsubscribe info
func (ca *ClusterAdapter) TriggerClusterDel(clusterName string) {
	log.DefaultLogger.Debugf("Delete Cluster %s", clusterName)
	ca.clusterMng.RemovePrimaryCluster(clusterName)
}
