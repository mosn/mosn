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

import "github.com/alibaba/sentinel-golang/core/hotspot/cache"

const (
	ConcurrencyMaxCount = 4000
	ParamsCapacityBase  = 4000
	ParamsMaxCapacity   = 20000
)

// ParamsMetric carries real-time counters for frequent ("hot spot") parameters.
//
// For each cache map, the key is the parameter value, while the value is the counter.
type ParamsMetric struct {
	// RuleTimeCounter records the last added token timestamp.
	RuleTimeCounter cache.ConcurrentCounterCache
	// RuleTokenCounter records the number of tokens.
	RuleTokenCounter cache.ConcurrentCounterCache
	// ConcurrencyCounter records the real-time concurrency.
	ConcurrencyCounter cache.ConcurrentCounterCache
}
