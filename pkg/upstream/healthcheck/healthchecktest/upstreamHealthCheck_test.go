package healthchecktest

import (
	"testing"
	"time"
	
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
)
var c  *config.MOSNConfig

func init(){
	c = config.Load("../../../../resource/mosn_config.json")
}

func Test_cluster_refreshHealthHosts(t *testing.T) {
	
	//parse cluster all in one
	clusters,clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	
	//create cluster manager
	cluster.NewClusterManager(nil, clusters, clusterMap,c.ClusterManager.AutoDiscovery,
		c.ClusterManager.RegistryUseHealthCheck)
	
	time.Sleep(1*time.Hour)
}