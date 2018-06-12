package types

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
)

// Health check interfaces

type FailureType string

const (
	FailureNetwork FailureType = "Network"
	FailurePassive FailureType = "Passive"
	FailureActive  FailureType = "Active"
)

type HealthCheckCb func(host Host, changedState bool)

// A health checker for an upstream cluster
type HealthChecker interface {
	// Start starts health checking, which will continually monitor hosts in upstream cluster
	Start()

	// Stop stops cluster health check. Client can use it to start/stop health check as a heartbeat
	Stop()

	// Add a health check callback, which will be called on a check round-trip is completed for a specified host.
	AddHostCheckCompleteCb(cb HealthCheckCb)

	// Used to update cluster's hosts for health checking
	OnClusterMemberUpdate(hostsAdded []Host, hostDel []Host)

	// Add cluster to healthChecker
	SetCluster(cluster Cluster)
}

// A health check session for an upstream host
type HealthCheckSession interface {
	// Start starts host health check
	Start()

	// Stop stops host health check
	Stop()

	// Set session as unhealthy for a specified reason
	SetUnhealthy(fType FailureType)
}

type HealthCheckHostMonitor interface {
}

var HealthCheckFactoryInstance HealthCheckerFactory

type HealthCheckerFactory interface {
	New(config v2.HealthCheck) HealthChecker
}
