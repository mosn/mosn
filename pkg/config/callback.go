package config

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

func (config *MOSNConfig) OnUpdateListeners(listeners []*pb.Listener)  error {
	for _, listener := range listeners {
		log.DefaultLogger.Infof("listener: %+v\n", listener)
	}
	return nil
}

func (config *MOSNConfig) OnUpdateRoutes(route *pb.RouteConfiguration) error {
	log.DefaultLogger.Infof("route: %+v\n", route)
	return nil
}

func (config *MOSNConfig) OnUpdateClusters(clusters []*pb.Cluster) error{
	for _, cluster := range clusters {
		log.DefaultLogger.Infof("cluster: %+v\n", cluster)
	}
	return nil
}

func (config *MOSNConfig) OnUpdateEndpoints(endpoints []*pb.ClusterLoadAssignment) error {
	for _, endpoint := range endpoints {
		log.DefaultLogger.Infof("endpoint: %+v\n", endpoint)
	}
	return nil
}
