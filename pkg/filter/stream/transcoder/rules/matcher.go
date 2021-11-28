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

package rules

import (
	"context"
	"mosn.io/mosn/pkg/types"
)

type RuleMatcher interface {
	Matches(ctx context.Context, headers types.HeaderMap) bool
}

type TransferRuleConfig struct {
	MatcherConfig *MatcherConfig `json:"macther_config"`
	RuleInfo      *RuleInfo      `json:"rule_info"`
}

type MatcherConfig struct {
	MatcherType string
	Config      interface{}
}

type RuleInfo struct {
	Type                string
	UpstreamProtocol    string                 `json:"upstream_protocol"`
	UpstreamSubProtocol string                 `json:"upstream_sub_protocol"`
	Description         string                 `json:"description"`
	Config              map[string]interface{} `json:"config"`
}

type TransferRule struct {
	Macther  RuleMatcher
	RuleInfo *RuleInfo
}
