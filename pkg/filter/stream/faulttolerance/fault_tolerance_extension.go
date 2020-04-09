package faulttolerance

import (
	v2 "mosn.io/mosn/pkg/config/v2"
)

var extensionGetMaxHostThresholdFunc map[string]func(*v2.FaultToleranceFilterConfig) uint64

func RegisterExtensionGetMaxHostThresholdFunc(name string, function func(*v2.FaultToleranceFilterConfig) uint64) {
	extensionGetMaxHostThresholdFunc[name] = function
}

func GetExtensionGetMaxHostThresholdFunc() map[string]func(*v2.FaultToleranceFilterConfig) uint64 {
	return extensionGetMaxHostThresholdFunc
}
