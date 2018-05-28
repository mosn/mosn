package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

var ClusterAdap ClusterAdapter

type ClusterAdapter struct {
	clusterMng *clusterManager
}

// called when confreg called
func (ca *ClusterAdapter) TriggerClusterUpdate(clusterName string, hosts []v2.Host) {
	log.DefaultLogger.Debugf("Update cluster,cluster name is : %s,hosts are: %+v", clusterName, hosts)
	clusterExist := ca.clusterMng.ClusterExist(clusterName)

	if !clusterExist {
		if ca.clusterMng.autoDiscovery {
			cluster := v2.Cluster{
				Name:           clusterName,
				ClusterType:    v2.DYNAMIC_CLUSTER,
				SubClustetType: v2.CONFREG_CLUSTER,
				LbType:         v2.LB_RANDOM,
			}
			ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
		} else {
			log.DefaultLogger.Debugf("[DEBUG] cluster:%s doesn't exist", clusterName)
			return
		}
	}
	
	log.DefaultLogger.Debugf("[TriggerClusterUpdate Called] cluster name is:%s hosts are:%+v",
		clusterName, hosts)
	ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
}

// called by service subscribe
func (ca *ClusterAdapter) TriggerClusterAdded(cluster v2.Cluster) {
	clusterExist := ca.clusterMng.ClusterExist(cluster.Name)
	
	if !clusterExist {
		log.DefaultLogger.Debugf("Add PrimaryCluster: %s", cluster.Name)
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
