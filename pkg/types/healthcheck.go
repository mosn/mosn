package types

type HealthCheckCb func(host Host, changedState bool)

type HealthChecker interface {
	AddHostCheckCompleteCb(cb HealthCheckCb)

	Start()
}

type HealthCheckHostMonitor interface {
}

