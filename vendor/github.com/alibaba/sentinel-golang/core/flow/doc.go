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

// Package flow implements the flow shaping control.
//
// flow module is based on QPS statistic metric
//
// The TrafficShapingController consists of two part: TrafficShapingCalculator and TrafficShapingChecker
//
//  1. TrafficShapingCalculator calculates the actual traffic shaping token threshold. Currently, Sentinel supports two token calculate strategy: Direct and WarmUp.
//  2. TrafficShapingChecker performs checking logic according to current metrics and the traffic shaping strategy, then yield the token result. Currently, Sentinel supports two control behavior: Reject and Throttling.
//
// Besides, Sentinel supports customized TrafficShapingCalculator and TrafficShapingChecker. User could call function SetTrafficShapingGenerator to register customized TrafficShapingController and call function RemoveTrafficShapingGenerator to unregister TrafficShapingController.
// There are a few notes users need to be aware of:
//
//  1. The function both SetTrafficShapingGenerator and RemoveTrafficShapingGenerator is not thread safe.
//  2. Users can not override the Sentinel supported TrafficShapingController.
//
package flow
