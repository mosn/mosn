/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

type FaultToleranceRuleJson struct {
	Enabled               bool     `json:"enabled"`
	ExceptionTypes        []uint32 `json:"exceptionTypes"`
	TimeWindow            int64    `json:"timeWindow"`
	LeastWindowCount      int64    `json:"leastWindowCount"`
	ExceptionRateMultiple float64  `json:"exceptionRateMultiple"`
	MaxIpCount            uint64   `json:"maxIpCount"`
	MaxIpRatio            float64  `json:"maxIpRatio"`
	RecoverTime           int64    `json:"recoverTime"`
	TaskSize              int64    `json:"taskSize"`
}
