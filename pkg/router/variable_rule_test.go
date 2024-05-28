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

package router

import (
	"context"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/variable"
)

func TestVariableRouteRuleImpl1(t *testing.T) {
	variable.Register(variable.NewStringVariable("header", nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable("method", nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable("uri", nil, nil, variable.DefaultStringSetter, 0))

	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		name     string
		value    string
		expected bool
	}{
		{"header", "test", true},
		{"header", "/test/test", false},
		{"method", "test", false},
		{"uri", "/1234", true},
		{"uri", "/abc", false},
	}
	// header == test || (method == test && regex.MatchString(uri)) || uri == /1234
	route := &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{
				Variables: []v2.VariableMatcher{
					{
						Name:  "header",
						Value: "test",
						Model: "or",
					},
					{
						Name:  "method",
						Value: "test",
						Model: "and",
					},
					{
						Name:  "uri",
						Regex: "/[0-9]+",
						Model: "or",
					},
					{
						Name:  "uri",
						Value: "/1234",
					},
				},
			},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: "test",
				},
			},
		},
	}
	for i, tc := range testCases {
		base, _ := NewRouteRuleImplBase(virtualHostImpl, route)
		variables := make([]*VariableMatchItem, len(route.Match.Variables))
		for i := range route.Match.Variables {
			variables[i] = ParseToVariableMatchItem(route.Match.Variables[i])
		}

		ctx := variable.NewVariableContext(context.Background())
		rr := &VariableRouteRuleImpl{base, variables}
		variable.SetString(ctx, tc.name, tc.value)
		result := rr.Match(ctx, protocol.CommonHeader(map[string]string{}))
		assert.EqualValuesf(t, result != nil, tc.expected, "#%d want matched %v, but get matched %v\n", i, tc.expected, result != nil)
		if result != nil {
			assert.EqualValuesf(t, api.Variable, result.RouteRule().PathMatchCriterion().MatchType(), "#%d match type is not expected", i)
		}
	}
}

func TestVariableRouteRuleImpl2(t *testing.T) {
	variable.Register(variable.NewStringVariable("header", nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable("method", nil, nil, variable.DefaultStringSetter, 0))
	variable.Register(variable.NewStringVariable("uri", nil, nil, variable.DefaultStringSetter, 0))

	virtualHostImpl := &VirtualHostImpl{virtualHostName: "test"}
	testCases := []struct {
		names    []string
		values   []string
		expected bool
	}{
		{[]string{"header", "method"}, []string{"test", "test1"}, true},
		{[]string{"header", "method"}, []string{"/test/test", "test1"}, false},
		{[]string{"header", "method"}, []string{"test", "test2"}, false},
		{[]string{"header", "uri"}, []string{"test3", "/1234"}, true},
		{[]string{"uri"}, []string{"/33"}, true},
	}
	// (header == test && method == test) || regex.MatchString(uri)) || uri == /1234
	route := &v2.Router{
		RouterConfig: v2.RouterConfig{
			Match: v2.RouterMatch{
				Variables: []v2.VariableMatcher{
					{
						Name:  "header",
						Value: "test",
						Model: "and",
					},
					{
						Name:  "method",
						Value: "test1",
						Model: "or",
					},
					{
						Name:  "uri",
						Regex: "/[0-9]+",
						Model: "or",
					},
					{
						Name:  "uri",
						Value: "/1234",
					},
				},
			},
			Route: v2.RouteAction{
				RouterActionConfig: v2.RouterActionConfig{
					ClusterName: "test",
				},
			},
		},
	}
	for i, tc := range testCases {
		base, _ := NewRouteRuleImplBase(virtualHostImpl, route)
		variables := make([]*VariableMatchItem, len(route.Match.Variables))
		for i := range route.Match.Variables {
			variables[i] = ParseToVariableMatchItem(route.Match.Variables[i])
		}

		ctx := variable.NewVariableContext(context.Background())
		rr := &VariableRouteRuleImpl{base, variables}
		for i := 0; i < len(tc.names); i++ {
			variable.SetString(ctx, tc.names[i], tc.values[i])
		}
		result := rr.Match(ctx, protocol.CommonHeader(map[string]string{}))
		assert.EqualValuesf(t, result != nil, tc.expected, "#%d want matched %v, but get matched %v\n", i, tc.expected, result != nil)
		if result != nil {
			assert.EqualValuesf(t, api.Variable, result.RouteRule().PathMatchCriterion().MatchType(), "#%d match type is not expected", i)
		}
	}
}
func TestParseToVariableMatchItem(t *testing.T) {
	s := func(s string) *string { return &s }
	reg, _ := regexp.Compile("/[0-9]+")
	type args struct {
		matcher v2.VariableMatcher
	}
	tests := []struct {
		name string
		args args
		want *VariableMatchItem
	}{
		{
			name: "p1",
			args: args{
				matcher: v2.VariableMatcher{
					Name:  "p1",
					Value: "test1",
				},
			},
			want: &VariableMatchItem{
				name:         "p1",
				value:        s("test1"),
				regexPattern: nil,
				model:        AND,
			},
		},
		{
			name: "p2",
			args: args{
				matcher: v2.VariableMatcher{
					Name:  "p2",
					Value: "test2",
					Model: "Or",
				},
			},
			want: &VariableMatchItem{
				name:         "p2",
				value:        s("test2"),
				regexPattern: nil,
				model:        OR,
			},
		},
		{
			name: "p3",
			args: args{
				matcher: v2.VariableMatcher{
					Name:  "p3",
					Model: "and",
					Regex: "/[0-9]+",
				},
			},
			want: &VariableMatchItem{
				name:         "p3",
				regexPattern: reg,
				model:        AND,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseToVariableMatchItem(tt.args.matcher); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseToVariableMatchItem() = %v, want %v", got, tt.want)
			}
		})
	}
}
