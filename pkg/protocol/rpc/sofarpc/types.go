package sofarpc

import (
	"time"
	apiv2 "github.com/alipay/sofa-mosn/pkg/api/v2"
	"errors"
)

const (
	// Encode/Decode Exception Msg
	UnKnownCmdType string = "unknown cmd type"
	UnKnownCmdCode string = "unknown cmd code"

	// Sofa Rpc Default HC Parameters
	SofaRPC                             = "SofaRpc"
	DefaultBoltHeartBeatTimeout         = 6 * 15 * time.Second
	DefaultBoltHeartBeatInterval        = 15 * time.Second
	DefaultIntervalJitter               = 5 * time.Millisecond
	DefaultHealthyThreshold      uint32 = 2
	DefaultUnhealthyThreshold    uint32 = 2
)

var (
	// Encode/Decode Exception
	ErrUnKnownCmdType = errors.New(UnKnownCmdType)
	ErrUnKnownCmdCode = errors.New(UnKnownCmdCode)
)

// DefaultSofaRPCHealthCheckConf
var DefaultSofaRPCHealthCheckConf = apiv2.HealthCheck{
	HealthCheckConfig: apiv2.HealthCheckConfig{
		Protocol:           SofaRPC,
		HealthyThreshold:   DefaultHealthyThreshold,
		UnhealthyThreshold: DefaultUnhealthyThreshold,
	},
	Timeout:        DefaultBoltHeartBeatTimeout,
	Interval:       DefaultBoltHeartBeatInterval,
	IntervalJitter: DefaultIntervalJitter,
}