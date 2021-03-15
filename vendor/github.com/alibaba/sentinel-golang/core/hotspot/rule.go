// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hotspot

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// ControlBehavior indicates the traffic shaping behaviour.
type ControlBehavior int32

const (
	Reject ControlBehavior = iota
	Throttling
)

func (t ControlBehavior) String() string {
	switch t {
	case Reject:
		return "Reject"
	case Throttling:
		return "Throttling"
	default:
		return strconv.Itoa(int(t))
	}
}

// MetricType represents the target metric type.
type MetricType int32

const (
	// Concurrency represents concurrency count.
	Concurrency MetricType = iota
	// QPS represents request count per second.
	QPS
)

func (t MetricType) String() string {
	switch t {
	case Concurrency:
		return "Concurrency"
	case QPS:
		return "QPS"
	default:
		return "Undefined"
	}
}

// Rule represents the hotspot(frequent) parameter flow control rule
type Rule struct {
	// ID is the unique id
	ID string `json:"id,omitempty"`
	// Resource is the resource name
	Resource string `json:"resource"`
	// MetricType indicates the metric type for checking logic.
	// For Concurrency metric, hotspot module will check the each hot parameter's concurrency,
	//		if concurrency exceeds the Threshold, reject the traffic directly.
	// For QPS metric, hotspot module will check the each hot parameter's QPS,
	//		the ControlBehavior decides the behavior of traffic shaping controller
	MetricType MetricType `json:"metricType"`
	// ControlBehavior indicates the traffic shaping behaviour.
	// ControlBehavior only takes effect when MetricType is QPS
	ControlBehavior ControlBehavior `json:"controlBehavior"`
	// ParamIndex is the index in context arguments slice.
	// if ParamIndex is great than or equals to zero, ParamIndex means the <ParamIndex>-th parameter
	// if ParamIndex is the negative, ParamIndex means the reversed <ParamIndex>-th parameter
	ParamIndex int `json:"paramIndex"`
	// ParamKey is the key in context attachments map.
	// ParamKey can be used as a supplement to ParamIndex to facilitate rules to quickly obtain parameter from a large number of parameters
	ParamKey string `json:"paramKey"`
	// Threshold is the threshold to trigger rejection
	Threshold int64 `json:"threshold"`
	// MaxQueueingTimeMs only takes effect when ControlBehavior is Throttling and MetricType is QPS
	MaxQueueingTimeMs int64 `json:"maxQueueingTimeMs"`
	// BurstCount is the silent count
	// BurstCount only takes effect when ControlBehavior is Reject and MetricType is QPS
	BurstCount int64 `json:"burstCount"`
	// DurationInSec is the time interval in statistic
	// DurationInSec only takes effect when MetricType is QPS
	DurationInSec int64 `json:"durationInSec"`
	// ParamsMaxCapacity is the max capacity of cache statistic
	ParamsMaxCapacity int64 `json:"paramsMaxCapacity"`
	// SpecificItems indicates the special threshold for specific value
	SpecificItems map[interface{}]int64 `json:"specificItems"`
	// Mode Switch
	Mode Mode `json:"mode"`
}

// Mode represents the switch.
type Mode string

// defines the mode enum value
const (
	MONITOR Mode = "MONITOR"
	CONTROL Mode = "CONTROL"
	CLOSE   Mode = "CLOSE"
)

func (r *Rule) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		// Return the fallback string
		return fmt.Sprintf("{Id:%s, Resource:%s, MetricType:%+v, ControlBehavior:%+v, ParamIndex:%d, Threshold:%d, MaxQueueingTimeMs:%d, BurstCount:%d, DurationInSec:%d, ParamsMaxCapacity:%d, SpecificItems:%+v}",
			r.ID, r.Resource, r.MetricType, r.ControlBehavior, r.ParamIndex, r.Threshold, r.MaxQueueingTimeMs, r.BurstCount, r.DurationInSec, r.ParamsMaxCapacity, r.SpecificItems)
	}
	return string(b)
}
func (r *Rule) ResourceName() string {
	return r.Resource
}

// IsStatReusable checks whether current rule is "statistically" equal to the given rule.
func (r *Rule) IsStatReusable(newRule *Rule) bool {
	return r.Resource == newRule.Resource && r.ControlBehavior == newRule.ControlBehavior && r.ParamsMaxCapacity == newRule.ParamsMaxCapacity && r.DurationInSec == newRule.DurationInSec && r.MetricType == newRule.MetricType
}

// Equals checks whether current rule is consistent with the given rule.
func (r *Rule) Equals(newRule *Rule) bool {
	baseCheck := r.Resource == newRule.Resource && r.MetricType == newRule.MetricType && r.ControlBehavior == newRule.ControlBehavior && r.ParamsMaxCapacity == newRule.ParamsMaxCapacity && r.ParamIndex == newRule.ParamIndex && r.Threshold == newRule.Threshold && r.DurationInSec == newRule.DurationInSec && reflect.DeepEqual(r.SpecificItems, newRule.SpecificItems)
	if !baseCheck {
		return false
	}
	if r.ControlBehavior == Reject {
		return r.BurstCount == newRule.BurstCount
	} else if r.ControlBehavior == Throttling {
		return r.MaxQueueingTimeMs == newRule.MaxQueueingTimeMs
	} else {
		return false
	}
}
