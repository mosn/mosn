package xds

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

type Callbacks struct {

}

func (callbacks *Callbacks) OnUpdateListeners(listeners []*pb.Listener) {
	for _, listener := range listeners {
		log.DefaultLogger.Infof("listener: %+v\n", listener)
	}
}

func (callbacks *Callbacks) OnUpdateRoutes(route *pb.RouteConfiguration) {
	log.DefaultLogger.Infof("route: %+v\n", route)
}

func (callbacks *Callbacks) OnUpdateClusters(clusters []*pb.Cluster) {
	for _, cluster := range clusters {
		log.DefaultLogger.Infof("cluster: %+v\n", cluster)
	}
}

func (callbacks *Callbacks) OnUpdateEndpoints(endpoints []*pb.ClusterLoadAssignment) {
	for _, endpoint := range endpoints {
		log.DefaultLogger.Infof("endpoint: %+v\n", endpoint)
	}
}