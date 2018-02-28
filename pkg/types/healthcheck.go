package types

type HealthChecker interface {
	AddHostCheckCompleteCb(cb func(host Host, changedState bool))

	Start()
}

type HealthCheckHostMonitor interface {
}

