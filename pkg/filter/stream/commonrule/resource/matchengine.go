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

package resource

import (
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// MatcherEngine match
type MatcherEngine struct {
	matcher Matcher
}

// NewMatcherEnine new
func NewMatcherEnine(matcher Matcher) MatcherEngine {
	m := MatcherEngine{
		matcher: matcher,
	}
	if m.matcher == nil {
		m.matcher = NewDefaultMatcher()
	}
	return m
}

func (e MatcherEngine) registryResourceMatcher(matcher Matcher) {
	e.matcher = matcher
}

// Match match
func (e MatcherEngine) Match(headers types.HeaderMap, ruleConfig *model.RuleConfig) bool {
	for _, resourceConfig := range ruleConfig.ResourceConfigs {
		if e.matcher.Match(headers, &resourceConfig) {
			return true
		}
	}
	return false
}
