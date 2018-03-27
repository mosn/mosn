package healthcheck

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.HealthChecker
type healthChecker struct {
	healthCheckCb types.HealthCheckCb
	healthCheckSessions map[string]*healthCheckSession
}


type healthCheckSession struct {

}