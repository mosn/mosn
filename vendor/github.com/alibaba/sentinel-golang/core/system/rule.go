package system

import (
	"encoding/json"
	"fmt"
)

type MetricType uint32

const (
	// Load represents system load1 in Linux/Unix.
	Load MetricType = iota
	// AvgRT represents the average response time of all inbound requests.
	AvgRT
	// Concurrency represents the concurrency of all inbound requests.
	Concurrency
	InboundQPS
	CpuUsage
	// MetricTypeSize indicates the enum size of MetricType.
	MetricTypeSize
)

func (t MetricType) String() string {
	switch t {
	case Load:
		return "load"
	case AvgRT:
		return "avgRT"
	case Concurrency:
		return "concurrency"
	case InboundQPS:
		return "inboundQPS"
	case CpuUsage:
		return "cpuUsage"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

type AdaptiveStrategy int32

const (
	NoAdaptive AdaptiveStrategy = -1
	BBR        AdaptiveStrategy = iota
)

func (t AdaptiveStrategy) String() string {
	switch t {
	case NoAdaptive:
		return "none"
	case BBR:
		return "bbr"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

type SystemRule struct {
	ID uint64 `json:"id,omitempty"`

	MetricType   MetricType       `json:"metricType"`
	TriggerCount float64          `json:"count"`
	Strategy     AdaptiveStrategy `json:"adaptiveStrategy"`
}

func (r *SystemRule) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		// Return the fallback string
		return fmt.Sprintf("SystemRule{metricType=%s, triggerCount=%.2f, adaptiveStrategy=%s}",
			r.MetricType, r.TriggerCount, r.Strategy)
	}
	return string(b)
}

func (r *SystemRule) ResourceName() string {
	return r.MetricType.String()
}
