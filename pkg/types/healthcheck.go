package types

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	
	"time"
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
	AddCluster(cluster Cluster)
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

// Sofa Rpc Default HC Parameters
const (
	SofaRpc                      string        = "SofaRpc"
	HealthName                   string        = "ToConfReg"
	DefaultBoltHeartBeatTimeout  time.Duration = 6 * 15 * time.Second
	DefaultBoltHeartBeatInterval time.Duration = 15 * time.Second
)

// Global HC Parameters
const (
	DefaultIntervalJitter     time.Duration = 5 * time.Millisecond
	DefaultHealthyThreshold   uint32        = 2
	DefaultUnhealthyThreshold uint32        = 2
)

var DefaultSofaRpcHealthCheckConf = v2.HealthCheck{
	Protocol:           SofaRpc,
	Timeout:            DefaultBoltHeartBeatTimeout,
	HealthyThreshold:   DefaultHealthyThreshold,
	UnhealthyThreshold: DefaultUnhealthyThreshold,
	Interval:           DefaultBoltHeartBeatInterval,
	IntervalJitter:     DefaultIntervalJitter,
}

var HealthCheckInss HealthCheckInsInterface

type HealthCheckInsInterface interface {
	NewHealthCheck(config v2.HealthCheck) HealthChecker
}
