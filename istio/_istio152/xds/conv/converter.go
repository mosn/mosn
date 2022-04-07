package conv

import (
	"sync"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// Converter as an interface for mock test
type Converter interface {
	Stats() *XdsStats
	AppendRouterName(name string)
	GetRouterNames() []string
	ConvertAddOrUpdateRouters(routers []*envoy_api_v2.RouteConfiguration)
	ConvertAddOrUpdateListeners(listeners []*envoy_api_v2.Listener)
	ConvertDeleteListeners(listeners []*envoy_api_v2.Listener)
	ConvertUpdateClusters(clusters []*envoy_api_v2.Cluster)
	ConvertDeleteClusters(clusters []*envoy_api_v2.Cluster)
	ConvertUpdateEndpoints(loadAssignments []*envoy_api_v2.ClusterLoadAssignment) error
}

type xdsConverter struct {
	stats XdsStats
	// rdsrecords stores the router config from router discovery
	rdsrecords map[string]struct{}
	mu         sync.Mutex
}

func NewConverter() *xdsConverter {
	return &xdsConverter{
		stats:      NewXdsStats(),
		rdsrecords: map[string]struct{}{},
	}
}

func (cvt *xdsConverter) Stats() *XdsStats {
	return &cvt.stats
}

func (cvt *xdsConverter) AppendRouterName(name string) {
	cvt.mu.Lock()
	defer cvt.mu.Unlock()
	cvt.rdsrecords[name] = struct{}{}
}

func (cvt *xdsConverter) GetRouterNames() []string {
	cvt.mu.Lock()
	defer cvt.mu.Unlock()
	names := make([]string, 0, len(cvt.rdsrecords))
	for name := range cvt.rdsrecords {
		names = append(names, name)
	}
	return names
}
