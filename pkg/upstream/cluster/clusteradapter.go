package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

var ClusterAdap ClusterAdapter

type ClusterAdapter struct {
	clusterMng *clusterManager
}

func (ca *ClusterAdapter) TriggerClusterUpdate(clusterName string, hosts []v2.Host) {
	log.DefaultLogger.Debugf("[DEBUG INFO]Update cluster,cluster name is : %s,hosts are: %+v", clusterName, hosts)

	//get clusterName
	//clusterName := ca.GetClusterNameByServiceName(serviceName)
	//   serviceName -> cluster -> route (== )
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
	
	//add hosts to existing cluster
	log.DefaultLogger.Debugf("[TriggerClusterUpdate Called] cluster name is:%s hosts are:%+v",
		clusterName,hosts)
	ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
}

func (ca *ClusterAdapter) TriggerClusterDel(clusterName string) {
	log.DefaultLogger.Debugf("TriggerClusterDel", clusterName)
	
	//get clusterName
	//remove hosts from existing cluster
	ca.clusterMng.RemovePrimaryCluster(clusterName)
}