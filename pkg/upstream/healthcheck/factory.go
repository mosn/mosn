package healthcheck

import (
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var sessionFactories map[types.Protocol]types.HealthCheckSessionFactory

func init() {
	sessionFactories = make(map[types.Protocol]types.HealthCheckSessionFactory)
}

func RegisterSessionFactory(p types.Protocol, f types.HealthCheckSessionFactory) {
	sessionFactories[p] = f
}

// CreateHealthCheck is a extendable function that can create different health checker
// by different health check session.
// The Default session is TCPDial session
func CreateHealthCheck(cfg v2.HealthCheck, cluster types.Cluster) types.HealthChecker {
	f, ok := sessionFactories[types.Protocol(cfg.Protocol)]
	if !ok {
		// not registered, use default session factory
		f = &TCPDialSessionFactory{}
	}
	return newHealthChecker(cfg, cluster, f)
}
