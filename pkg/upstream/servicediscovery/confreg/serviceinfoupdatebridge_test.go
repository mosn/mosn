package registry

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
	"testing"
)

var c *config.MOSNConfig

func init() {
	c = config.Load("../../../../../mosn/resource/mosn_config.json")
	config.ConfigPath = "../../../../../mosn/resource/mosn_config_dump_result.json"
}

func TestGetAppInfoFromConfig(t *testing.T) {
	//parse service registry info
	config.ParseClusterConfig(c.ClusterManager.Clusters)
	fmt.Println(GetSubListFromConfig())

	config.ParseServiceRegistry(c.ServiceRegistry)
	fmt.Println(GetAppInfoFromConfig())
}

func TestGetSubListFromConfig(t *testing.T) {
	config.ParseClusterConfig(c.ClusterManager.Clusters)
	fmt.Println(GetSubListFromConfig())
}

func TestGetPubListFromConfig(t *testing.T) {
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery)

	config.ParseServiceRegistry(c.ServiceRegistry)
	fmt.Println(GetPubListFromConfig())
}

func TestResetApplicationInfo(t *testing.T) {
	config.ParseClusterConfig(c.ClusterManager.Clusters)
	config.ParseServiceRegistry(c.ServiceRegistry)
	app := v2.ApplicationInfo{
		AntShareCloud: true,
		DataCenter:    "dc2",
		AppName:       "testResetApplicationInfo",
	}
	fmt.Println(ResetApplicationInfo(app))
}

func TestResetRegistryInfo(t *testing.T) {
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery)

	config.ParseServiceRegistry(c.ServiceRegistry)
	app := v2.ApplicationInfo{
		AntShareCloud: true,
		DataCenter:    "dc2",
		AppName:       "testResetApplicationInfo",
	}

	//create cluster manager
	ResetRegistryInfo(app)
}

func TestAddPubInfo(t *testing.T) {
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery)

	config.ParseServiceRegistry(c.ServiceRegistry)
	m := map[string]string{"serviceNameAdded": "DataAdded"}
	AddPubInfo(m)
}

func TestDelPubInfo(t *testing.T) {
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery)

	config.ParseServiceRegistry(c.ServiceRegistry)
	//m := map[string]string{"serviceNameAdded":"DataAdded"}
	DelPubInfo("pub_service_1")
}

func TestAddSubInfo(t *testing.T) {
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery)

	//config.ParseServiceRegistry(c.ServiceRegistry)
	s := []string{"testAddedSub"}
	AddSubInfo(s)
}

func TestDelSubInfo(t *testing.T) {
	clusters, clusterMap := config.ParseClusterConfig(c.ClusterManager.Clusters)
	cluster.NewClusterManager(nil, clusters, clusterMap, c.ClusterManager.AutoDiscovery)

	config.ParseServiceRegistry(c.ServiceRegistry)
	s := []string{"x_test_service2"}
	DelSubInfo(s)
}

func TestRegisterConfigParsedListener(t *testing.T) {
	config.RegisterConfigParsedListener(config.ParseCallbackKeyCluster, OnClusterInfoParsed)
}
