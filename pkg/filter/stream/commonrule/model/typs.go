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

package model

// RunMode
const (
	RunModeControl = "CONTROL"
	RunModeMonitor = "MONITOR"
)

// CommonRuleConfig config
type CommonRuleConfig struct {
	RuleConfigs []RuleConfig `json:"rule_configs"`
}

// RuleConfig config
type RuleConfig struct {
	Id      int    `json:"id"`
	Index   int    `json:"index"`
	Name    string `json:"name"`
	AppName string `json:"appName"`
	Enable  bool   `json:"enable"`
	RunMode string `json:"run_mode"`

	LimitConfig     LimitConfig      `json:"limit"`
	ResourceConfigs []ResourceConfig `json:"resources"`
}

// ComparisonCofig config
type ComparisonCofig struct {
	CompareType string `json:"compare_type"`
	Key         string `json:"key"`
	Value       string `json:"value"`
}

// ResourceConfig config
type ResourceConfig struct {
	Headers         []ComparisonCofig `json:"headers"`
	HeadersRelation string            `json:"headers_relation"`
	Params          []ComparisonCofig `json:"params"`
	ParamsRelation  string            `json:"params_relation"`
}

// LimitConfig config
type LimitConfig struct {
	LimitStrategy string  `json:"limit_strategy"`
	MaxBurstRatio float64 `json:"max_burst_ratio"`
	PeriodMs      int     `json:"period_ms"`
	MaxAllows     int     `json:"max_allows"`
}
