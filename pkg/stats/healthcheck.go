package stats

import "github.com/alipay/sofa-mosn/pkg/types"

// HeathCheckType represents health check metrics type
const HealthCheckType = "healthcheck"

// health check metrics key
const (
	HealthCheckAttempt        = "health_check_attempt"
	HealthCheckSuccess        = "health_check_success"
	HealthCheckFailure        = "health_check_failure"
	HealthCheckPassiveFailure = "health_check_passive_failure"
	HealthCheckNetworkFailure = "health_check_network_failure"
	HealthCheckVeirfyCluster  = "health_check_verify_cluster"
	HealthCheckHealthy        = "health_check_healty"
)

func NewHealthStats(servicename string) types.Metrics {
	return NewStats(HealthCheckType, servicename)
}
