package xds

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type Config struct {

}

func (config *Config) OnUpdateListeners(listeners []*pb.Listener) {
	for _, listener := range listeners {
		log.DefaultLogger.Infof("listener: %+v\n", listener)
	}
}

func (config *Config) OnUpdateRoutes(route *pb.RouteConfiguration) {
	log.DefaultLogger.Infof("route: %+v\n", route)
}

func (config *Config) OnUpdateClusters(clusters []*pb.Cluster) {
	for _, cluster := range clusters {
		log.DefaultLogger.Infof("cluster: %+v\n", cluster)
	}
}

func (config *Config) OnUpdateEndpoints(endpoints []*pb.ClusterLoadAssignment) {
	for _, endpoint := range endpoints {
		log.DefaultLogger.Infof("endpoint: %+v\n", endpoint)
	}
}