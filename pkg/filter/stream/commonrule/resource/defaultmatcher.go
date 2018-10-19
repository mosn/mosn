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
	"strings"

	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// DefaultMatcher macher
type DefaultMatcher struct {
}

// NewDefaultMatcher mew
func NewDefaultMatcher() *DefaultMatcher {
	return &DefaultMatcher{}
}

// Match match
func (m *DefaultMatcher) Match(headers types.HeaderMap, resourceConfig *model.ResourceConfig) bool {
	if resourceConfig.Headers != nil && len(resourceConfig.Headers) > 0 && !m.matchHeaders(headers, resourceConfig) {
		return false
	}

	value, _ := headers.Get(protocol.MosnHeaderQueryStringKey)
	if resourceConfig.Params != nil && len(resourceConfig.Params) > 0 && !m.matchQueryParams(value, resourceConfig) {
		return false
	}

	return true
}

func (*DefaultMatcher) matchHeaders(headers types.HeaderMap, resourceConfig *model.ResourceConfig) bool {
	matched := resourceConfig.ParamsRelation != RelationOr
	for _, comparison := range resourceConfig.Headers {
		value, _ := headers.Get(comparison.Key)
		flag := compare(comparison, value)
		if resourceConfig.ParamsRelation != RelationOr {
			matched = matched && flag
		} else {
			matched = matched || flag
		}
	}
	return matched
}

func (*DefaultMatcher) matchQueryParams(queryString string, resourceConfig *model.ResourceConfig) bool {
	ss := strings.Split(queryString, "&")
	queryParams := make(map[string][]string)
	for _, item := range ss {
		keyValue := strings.SplitN(item, "=", 2)
		if len(keyValue) == 2 {
			queryParams[keyValue[0]] = append(queryParams[keyValue[0]], keyValue[1])
		}
	}

	matched := resourceConfig.ParamsRelation != RelationOr
	for _, param := range resourceConfig.Params {
		flag := compares(param, queryParams[param.Key])
		if resourceConfig.ParamsRelation != RelationOr {
			matched = matched && flag
		} else {
			matched = matched || flag
		}
	}
	return matched
}

func compares(config model.ComparisonCofig, targets []string) bool {
	if config.CompareType == CompareEquals {
		for _, target := range targets {
			if config.Value == target {
				return true
			}
		}
		return false
	} else if config.CompareType == CompareNotEquals {
		for _, target := range targets {
			if config.Value == target {
				return false
			}
		}
		return true
	}
	return true
}

func compare(config model.ComparisonCofig, target string) bool {
	switch config.CompareType {
	case CompareEquals:
		return config.Value == target
	case CompareNotEquals:
		return config.Value != target
	default:
		return true
	}
}
