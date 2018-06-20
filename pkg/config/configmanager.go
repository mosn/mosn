package config

import (
	"fmt"
	
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)
// dumper provides basic operation with mosn elements, like 'cluster', to write back the config file with dynamic changes
// biz logic operation, like 'clear all subscribe info', should be written in the bridge code, not in config module.
//
// changes dump flow :
//
// biz ops -> bridge module -> config module
//
//  dumped info load flow:
//
// 1. bridge module register key of interesting config(like 'cluster') into config module
// 2. config parser invoke callback functions (if exists) of config key
// 3. bridge module get biz info(like service subscribe/publish, application info) from callback invocations
// 4. biz module(like confreg) get biz info from bridge module directly

//todo: parse & re-assemble in ConfregConfigSynchronizer @boqin
func ResetServiceRegistryInfo(appInfo v2.ApplicationInfo, subServiceList []string) {
	// reset service info
	config.ServiceRegistry.ServiceAppInfo = ServiceAppInfoConfig{
		AntShareCloud: appInfo.AntShareCloud,
		DataCenter:    appInfo.DataCenter,
		AppName:       appInfo.AppName,
	}
	
	// reset servicePubInfo
	config.ServiceRegistry.ServicePubInfo = []ServicePubInfoConfig{}
	
	// delete subInfo / dynamic clusters
	RemoveClusterConfig(subServiceList)
}


func AddClusterConfig(clusters []v2.Cluster) {
	for _, cluster := range clusters {
		clusterConfig := convertClusterConfig(cluster)
		exist := false
		
		for i, _ := range config.ClusterManager.Clusters {
			// rewrite cluster's info if exist already
			if config.ClusterManager.Clusters[i].Name == clusterConfig.Name {
				config.ClusterManager.Clusters[i] = clusterConfig
				exist = true
				break
			}
		}
		
		//added cluster if not exist
		if !exist {
			config.ClusterManager.Clusters = append(config.ClusterManager.Clusters, clusterConfig)
		}
		
		// update routes
		//AddRouterConfig(cluster.Name)
	}
	go Dump(true)
}

func RemoveClusterConfig(clusterNames []string) {
	dirty := false
	
	for _, clusterName := range clusterNames {
		for i, cluster := range config.ClusterManager.Clusters {
			if cluster.Name == clusterName {
				//remove
				config.ClusterManager.Clusters = append(config.ClusterManager.Clusters[:i], config.ClusterManager.Clusters[i+1:]...)
				dirty = true
				break
			}
		}
	}
	
	go Dump(dirty)
}

func AddPubInfo(pubInfoAdded map[string]string) {
	for srvName, srvData := range pubInfoAdded {
		exist := false
		srvPubInfo := ServicePubInfoConfig{
			ServiceName: srvName,
			PubData:     srvData,
		}
		
		for i, _ := range config.ServiceRegistry.ServicePubInfo {
			// rewrite cluster's info
			if config.ServiceRegistry.ServicePubInfo[i].ServiceName == srvName {
				config.ServiceRegistry.ServicePubInfo[i] = srvPubInfo
				exist = true
				break
			}
		}
		
		if !exist {
			config.ServiceRegistry.ServicePubInfo = append(config.ServiceRegistry.ServicePubInfo, srvPubInfo)
		}
	}
	
	go Dump(true)
}

func DelPubInfo(serviceName string) {
	dirty := false
	
	for i, srvPubInfo := range config.ServiceRegistry.ServicePubInfo {
		if srvPubInfo.ServiceName == serviceName {
			//remove
			config.ServiceRegistry.ServicePubInfo = append(config.ServiceRegistry.ServicePubInfo[:i], config.ServiceRegistry.ServicePubInfo[i+1:]...)
			dirty = true
			break
		}
	}
	
	go Dump(dirty)
}

// ~ convert functions, api.v2 model -> config model
func convertClusterConfig(cluster v2.Cluster) ClusterConfig {
	return ClusterConfig{
		Name:                 cluster.Name,
		Type:                 string(cluster.ClusterType),
		SubType:              string(cluster.SubClustetType),
		LbType:               string(cluster.LbType),
		CircuitBreakers:      cluster.CirBreThresholds,
		//HealthCheck        : nil,
		ClusterSpecConfig: convertClusterSpec(cluster.Spec),
	}
}

func convertClusterSpec(clusterSpec v2.ClusterSpecInfo) ClusterSpecConfig {
	var specs []SubscribeSpecConfig
	
	for _, sub := range clusterSpec.Subscribes {
		specs = append(specs, SubscribeSpecConfig{
			ServiceName: sub.ServiceName,
		})
	}
	
	return ClusterSpecConfig{
		Subscribes: specs,
	}
}

//// todo: add router config delete
func AddRouterConfig(clusterName string) {
	routerName := clusterName[0 : len(clusterName)-8]

	for _, l := range config.Servers[0].Listeners {
		if routers, ok := l.FilterChains[0].Filters[0].Config["routes"].([]interface{}); ok {
			// remove repetition
			for _, route := range routers {
				if r, ok := route.(map[string]interface{}); ok {
					if n, ok := r["name"].(string); ok {
						if n == routerName {
							return
						}
					}
				}
			}

			// append router
			var s = make(map[string]interface{}, 4)
			s["name"] = routerName
			s["service"] = routerName
			s["cluster"] = clusterName
			routers = append(routers, s)
			l.FilterChains[0].Filters[0].Config["routes"] = routers
		} else {
			fmt.Println(l.FilterChains[0].Filters[0].Config["routes"])
		}
	}
}
