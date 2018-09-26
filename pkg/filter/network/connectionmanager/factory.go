package connectionmanager

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/filter"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// todo this filter may use in the future
func init() {
	filter.RegisterNetwork(v2.CONNECTION_MANAGER, CreateProxyFactory)
}

type connectionManagerFilterConfigFactory struct {
}

func (cmfcf *connectionManagerFilterConfigFactory) CreateFilterChain(context context.Context, clusterManager types.ClusterManager, callbacks types.NetWorkFilterChainFactoryCallbacks) {

}

func CreateProxyFactory(conf map[string]interface{}) (types.NetworkFilterChainFactory, error) {
	return &connectionManagerFilterConfigFactory{}, nil
}
