package tunnel

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/upstream/cluster"
)

type HostRemover struct {
	address string
	cluster string
}

func NewHostRemover(address string, cluster string) *HostRemover {
	return &HostRemover{address: address, cluster: cluster}
}

func (h *HostRemover) OnEvent(event api.ConnectionEvent) {
	if event.IsClose() {
		log.DefaultLogger.Infof("try to remove host: %v in cluster: %v", h.address, h.cluster)
		cluster.GetClusterMngAdapterInstance().ClusterManager.RemoveClusterHosts(h.cluster, []string{h.address})
	}
}
