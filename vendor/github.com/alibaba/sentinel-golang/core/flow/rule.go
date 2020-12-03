package flow

import (
	"encoding/json"
	"fmt"
)

const (
	// LimitOriginDefault represents all origins.
	LimitOriginDefault = "default"
	// LimitOriginOther represents all origins excluding those configured in other rules.
	// For example, if resource "abc" has a rule whose limit origin is "originA",
	// the "other" origin will represents all origins excluding "originA".
	LimitOriginOther = "other"
)

// MetricType represents the target metric type.
type MetricType int32

const (
	// Concurrency represents concurrency count.
	Concurrency MetricType = iota
	// QPS represents request count per second.
	QPS
)

// RelationStrategy indicates the flow control strategy based on the relation of invocations.
type RelationStrategy int32

const (
	// Direct means flow control by current resource directly.
	Direct RelationStrategy = iota
	// AssociatedResource means flow control by the associated resource rather than current resource.
	AssociatedResource
)

// ControlBehavior indicates the traffic shaping behaviour.
type ControlBehavior int32

const (
	Reject ControlBehavior = iota
	WarmUp
	Throttling
	WarmUpThrottling
)

type ClusterThresholdMode uint32

const (
	AvgLocalThreshold ClusterThresholdMode = iota
	GlobalThreshold
)

type ClusterRuleConfig struct {
	ThresholdType ClusterThresholdMode `json:"thresholdType"`
}

// FlowRule describes the strategy of flow control.
type FlowRule struct {
	// ID represents the unique ID of the rule (optional).
	ID uint64 `json:"id,omitempty"`

	// Resource represents the resource name.
	Resource string `json:"resource"`
	// LimitOrigin represents the target origin (reserved field).
	LimitOrigin string     `json:"limitApp"`
	MetricType  MetricType `json:"grade"`
	// Count represents the threshold.
	Count            float64          `json:"count"`
	RelationStrategy RelationStrategy `json:"strategy"`
	ControlBehavior  ControlBehavior  `json:"controlBehavior"`

	RefResource       string `json:"refResource,omitempty"`
	WarmUpPeriodSec   uint32 `json:"warmUpPeriodSec"`
	MaxQueueingTimeMs uint32 `json:"maxQueueingTimeMs"`
	// ClusterMode indicates whether the rule is for cluster flow control or local.
	ClusterMode   bool              `json:"clusterMode"`
	ClusterConfig ClusterRuleConfig `json:"clusterConfig"`
}

func (f *FlowRule) String() string {
	b, err := json.Marshal(f)
	if err != nil {
		// Return the fallback string
		return fmt.Sprintf("FlowRule{resource=%s, id=%d, metricType=%d, threshold=%.2f}",
			f.Resource, f.ID, f.MetricType, f.Count)
	}
	return string(b)
}

func (f *FlowRule) ResourceName() string {
	return f.Resource
}
