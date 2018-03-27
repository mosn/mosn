package types

type FailureType string

const (
	FailureNetwork FailureType = "Network"
	FailurePassive FailureType = "Passive"
	FailureActive FailureType = "Active"
)

type HealthCheckCb func(host Host, changedState bool)

type HealthChecker interface {
	Start()

	Stop()

	AddHostCheckCompleteCb(cb HealthCheckCb)
}

type HealthCheckSession interface {
	Start()

	Stop()

	SetUnhealthy(fType FailureType)
}

type HealthCheckHostMonitor interface {
}
