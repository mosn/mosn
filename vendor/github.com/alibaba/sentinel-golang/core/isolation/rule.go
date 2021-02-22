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

package isolation

import (
	"encoding/json"
	"fmt"
)

// MetricType represents the target metric type.
type MetricType int32

const (
	// Concurrency represents concurrency count.
	Concurrency MetricType = iota
)

func (s MetricType) String() string {
	switch s {
	case Concurrency:
		return "Concurrency"
	default:
		return "Undefined"
	}
}

// Rule describes the concurrency num control, that is similar to semaphore
type Rule struct {
	// ID represents the unique ID of the rule (optional).
	ID         string     `json:"id,omitempty"`
	Resource   string     `json:"resource"`
	MetricType MetricType `json:"metricType"`
	Threshold  uint32     `json:"threshold"`
}

func (r *Rule) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		// Return the fallback string
		return fmt.Sprintf("{Id=%s, Resource=%s, MetricType=%s, Threshold=%d}", r.ID, r.Resource, r.MetricType.String(), r.Threshold)
	}
	return string(b)
}

func (r *Rule) ResourceName() string {
	return r.Resource
}
