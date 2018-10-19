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
	RuleConfigs []RuleConfig
}

// RuleConfig config
type RuleConfig struct {
	Id      int    `json:"id"`
	Index   int    `json:"index"`
	Name    string `json:"name"`
	AppName string `json:"appName"`
	Enable  bool   `json:"enable"`
	RunMode string `json:"runMode"`

	LimitConfig     LimitConfig      `json:"limit"`
	ResourceConfigs []ResourceConfig `json:"resources"`
}

// ComparisonCofig config
type ComparisonCofig struct {
	CompareType string `json:"compareType"`
	Key         string `json:"key"`
	Value       string `json:"value"`
}

// ResourceConfig config
type ResourceConfig struct {
	Headers         []ComparisonCofig `json:"headers"`
	HeadersRelation string            `json:"headersRelation"`
	Params          []ComparisonCofig `json:"params"`
	ParamsRelation  string            `json:"paramsRelation"`
}

// LimitConfig config
type LimitConfig struct {
	LimitStrategy string  `json:"limitStrategy"`
	MaxBurstRatio float64 `json:"maxBurstRatio"`
	PeriodMs      int     `json:"periodMs"`
	MaxAllows     int     `json:"maxAllows"`
}
