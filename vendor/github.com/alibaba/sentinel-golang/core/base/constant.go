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

package base

// global variable
const (
	TotalInBoundResourceName = "__total_inbound_traffic__"

	DefaultMaxResourceAmount uint32 = 10000

	DefaultSampleCount uint32 = 2
	DefaultIntervalMs  uint32 = 1000

	// default 10*1000/500 = 20
	DefaultSampleCountTotal uint32 = 20
	// default 10s (total length)
	DefaultIntervalMsTotal uint32 = 10000

	DefaultStatisticMaxRt = int64(60000)
)
