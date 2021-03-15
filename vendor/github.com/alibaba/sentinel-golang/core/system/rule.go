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
	// 1
	BBR AdaptiveStrategy = iota
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

type Rule struct {
	ID           string           `json:"id,omitempty"`
	MetricType   MetricType       `json:"metricType"`
	TriggerCount float64          `json:"triggerCount"`
	Strategy     AdaptiveStrategy `json:"strategy"`
}

func (r *Rule) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		// Return the fallback string
		return fmt.Sprintf("Rule{metricType=%s, triggerCount=%.2f, adaptiveStrategy=%s}",
			r.MetricType, r.TriggerCount, r.Strategy)
	}
	return string(b)
}

func (r *Rule) ResourceName() string {
	return r.MetricType.String()
}
