package healthcheck

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	metrics "github.com/rcrowley/go-metrics"
)

type healthCheckStats struct {
	attempt metrics.Counter
	// total counts for check health returns ok
	success metrics.Counter
	// total counts for check health returns fail
	failure metrics.Counter
	// total counts for check health returns fail with reason
	passiveFailure metrics.Counter
	activeFailure  metrics.Counter
	networkFailure metrics.Counter
	verifyCluster  metrics.Counter
	healthy        metrics.Gauge
}

func newHealthCheckStats(namespace string) *healthCheckStats {
	m := stats.NewHealthStats(namespace)
	return &healthCheckStats{
		attempt:        m.Counter(stats.HealthCheckAttempt),
		success:        m.Counter(stats.HealthCheckSuccess),
		failure:        m.Counter(stats.HealthCheckFailure),
		activeFailure:  m.Counter(stats.HealthCheckActiveFailure),
		passiveFailure: m.Counter(stats.HealthCheckPassiveFailure),
		networkFailure: m.Counter(stats.HealthCheckNetworkFailure),
		verifyCluster:  m.Counter(stats.HealthCheckVeirfyCluster),
		healthy:        m.Gauge(stats.HealthCheckHealthy),
	}
}
