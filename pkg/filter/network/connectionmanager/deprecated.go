package connectionmanager

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

func init() {
	api.RegisterNetwork(v2.CONNECTION_MANAGER, CreateProxyFactory)
}

func CreateProxyFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	log.DefaultLogger.Warnf("deprecated filter. the filter connection_manager is removed, use routers in server instead.")
	return nil, nil
}
