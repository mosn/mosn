package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

var ClusterAdap ClusterAdapter

type ClusterAdapter struct {
	clusterMng             *clusterManager
}

func (ca *ClusterAdapter) TriggerClusterUpdate(serviceName string, hosts []v2.Host) {
	log.DefaultLogger.Debugf("[DEBUG INFO]Update cluster,cluster name is : %s,hosts are: %+v",serviceName,hosts)

	//get clusterName
	clusterName := ca.GetClusterNameByServiceName(serviceName)

	//update cluster
	clusterExist := ca.clusterMng.ClusterExist(clusterName)

	if !clusterExist {
		cluster := v2.Cluster {
			Name:           clusterName,
			ClusterType:    v2.DYNAMIC_CLUSTER,
			SubClustetType: v2.CONFREG_CLUSTER,
			LbType:         v2.LB_RANDOM,
		}
		//new cluster
		ca.clusterMng.AddOrUpdatePrimaryCluster(cluster)
	}

	//add hosts to cluster
	ca.clusterMng.UpdateClusterHosts(clusterName, 0, hosts)
}

func (ca *ClusterAdapter)GetClusterNameByServiceName(serviceName string) string {
	return serviceName
}