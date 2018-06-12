package healthcheck

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func init() {
	types.HealthCheckFactoryInstance = factory
}

var factory = &healthCheckerFactory{}

type healthCheckerFactory struct{}

func (nhc *healthCheckerFactory) New(config v2.HealthCheck) types.HealthChecker {
	switch config.Protocol {
	case string(protocol.SofaRpc):
		return newSofaRpcHealthChecker(config)
	case string(protocol.Http2):
		return newHttpHealthCheck(config)
		// todo: http1
	default:
		// todo: http1
		return nil
	}
}