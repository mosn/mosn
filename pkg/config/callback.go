package config

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

func (config *MOSNConfig) OnUpdateListeners(listeners []*pb.Listener)  error {
	for _, listener := range listeners {
		mosnListener := convertListenerConfig(listener)
		if mosnListener == nil {
			continue
		}
		//AddOrUpdateListener(mosnListener)
		log.DefaultLogger.Infof("listener: %+v\n", mosnListener)
	}
	return nil
}

/*
func (config *MOSNConfig) OnUpdateRoutes(route *pb.RouteConfiguration) error {
	log.DefaultLogger.Infof("route: %+v\n", route)
	return nil
}
*/

func (config *MOSNConfig) OnUpdateClusters(clusters []*pb.Cluster) error{
	mosnClusters := convertClustersConfig(clusters)
	//UpdateClusterConfig(mosnClusters)
	for _, cluster := range mosnClusters {
		log.DefaultLogger.Infof("cluster: %+v\n", cluster)
	}
	return nil
}

func (config *MOSNConfig) OnUpdateEndpoints(loadAssignments []*pb.ClusterLoadAssignment) error {
	for _, loadAssignment := range loadAssignments {
		for _, endpoints := range loadAssignment.Endpoints {
			hosts := convertEndpointsConfig(&endpoints)
			//UpdateClusterHost(loadAssignment.ClusterName, endpoints.Priority, hosts)
			for _, host := range hosts {
				log.DefaultLogger.Infof("endpoint: cluster: %s, priority: %d, %+v\n", loadAssignment.ClusterName, endpoints.Priority, host)
			}
		}
	}
	return nil
}
