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
	"sort"
	"testing"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

func TestNewMetadataMatchCriteriaImpl(t *testing.T) {
	type args struct {
		metadataMatches map[string]string
	}
	tests := []struct {
		name string
		args args
		want *MetadataMatchCriteriaImpl
	}{
		{
			name: "case4",
			args: args{
				metadataMatches: map[string]string{},
			},
			want: &MetadataMatchCriteriaImpl{},
		},
		{
			name: "case1",
			args: args{
				metadataMatches: map[string]string{
					"label": "green", "version": "v1", "appInfo": "test",
				},
			},
			want: &MetadataMatchCriteriaImpl{
				MatchCriteriaArray: []api.MetadataMatchCriterion{
					&MetadataMatchCriterionImpl{
						Name:  "appInfo",
						Value: "test",
					},
					&MetadataMatchCriterionImpl{
						Name:  "label",
						Value: "green",
					},
					&MetadataMatchCriterionImpl{
						Name:  "version",
						Value: "v1",
					},
				},
			},
		},
		{
			name: "case2",
			args: args{
				metadataMatches: map[string]string{
					"version": "v1", "appInfo": "test", "label": "green",
				},
			},
			want: &MetadataMatchCriteriaImpl{
				MatchCriteriaArray: []api.MetadataMatchCriterion{
					&MetadataMatchCriterionImpl{
						Name:  "appInfo",
						Value: "test",
					},
					&MetadataMatchCriterionImpl{
						Name:  "label",
						Value: "green",
					},
					&MetadataMatchCriterionImpl{
						Name:  "version",
						Value: "v1",
					},
				},
			},
		},
		{
			name: "case3",
			args: args{
				metadataMatches: map[string]string{
					"version": "v1",
				},
			},
			want: &MetadataMatchCriteriaImpl{
				MatchCriteriaArray: []api.MetadataMatchCriterion{
					&MetadataMatchCriterionImpl{
						Name:  "version",
						Value: "v1",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMetadataMatchCriteriaImpl(tt.args.metadataMatches); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMetadataMatchCriteriaImpl() = %v, want %v, case = %s", got, tt.want, tt.name)
			}
		})
	}
}

func Test_NewConfigImpl(t *testing.T) {

	type args struct {
		routerConfig *v2.RouterConfiguration
	}

	FALSE := false

	tests := []struct {
		name string
		args args
		want *configImpl
	}{
		{
			name: "case1",
			args: args{
				routerConfig: &v2.RouterConfiguration{
					RouterConfigurationConfig: v2.RouterConfigurationConfig{
						RequestHeadersToAdd: []*v2.HeaderValueOption{
							{
								Header: &v2.HeaderValue{
									Key:   "LEVEL",
									Value: "1",
								},
								Append: &FALSE,
							},
						},
						ResponseHeadersToAdd: []*v2.HeaderValueOption{
							{
								Header: &v2.HeaderValue{
									Key:   "Random",
									Value: "123456",
								},
								Append: &FALSE,
							},
						},
						RequestHeadersToRemove:  []string{"test"},
						ResponseHeadersToRemove: []string{"status"},
					},
				},
			},
			want: &configImpl{
				requestHeadersParser: &headerParser{
					headersToAdd: []*headerPair{
						{
							headerName: "level",
							headerFormatter: &plainHeaderFormatter{
								isAppend:    false,
								staticValue: "1",
							},
						},
					},
					headersToRemove: []string{"test"},
				},
				responseHeadersParser: &headerParser{
					headersToAdd: []*headerPair{
						{
							headerName: "random",
							headerFormatter: &plainHeaderFormatter{
								isAppend:    false,
								staticValue: "123456",
							},
						},
					},
					headersToRemove: []string{"status"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewConfigImpl(tt.args.routerConfig); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConfigImpl(routerConfig *v2.RouterConfiguration) = %v, want %v", got, tt.want)
			}
		})
	}
}

// test MetadataMatchCriteriaImpl's sort.Interface
func TestMetadataMatchCriteriaImplSort(t *testing.T) {
	keys := []string{"1", "3", "2", "0"}
	values := []string{"b", "d", "c", "a"}
	var mciArray []api.MetadataMatchCriterion
	for i := range keys {
		mmci := &MetadataMatchCriterionImpl{
			Name:  keys[i],
			Value: values[i],
		}
		mciArray = append(mciArray, mmci)
	}
	m := &MetadataMatchCriteriaImpl{mciArray}
	sort.Sort(m)
	expected := []string{"0", "1", "2", "3"}
	for i, mmci := range m.MatchCriteriaArray {
		if mmci.MetadataKeyName() != expected[i] {
			t.Error("sort unexpected")
		}
	}
}

func TestHTTPHeaderMatch(t *testing.T) {
	t.Run("simple http header match", func(t *testing.T) {
		headersConfig := []v2.HeaderMatcher{
			{
				Name:  "test-key",
				Value: "test-value",
			},
			{
				Name:  "key2",
				Value: "value2",
			},
		}
		// request headers must contains all of kvs
		// request header can contains kv not in the config
		matcher := CreateHTTPHeaderMatcher(headersConfig)
		for idx, c := range []struct {
			requestHeader map[string]string
			matched       bool
		}{
			{map[string]string{"test-key": "test-value", "key2": "value2"}, true},
			{map[string]string{"test-key": "test-value", "key2": "value2", "more": "more"}, true},
			{map[string]string{"test-key": "test-value"}, false},
			{map[string]string{"key2": "value2"}, false},
			{map[string]string{"test-key": "test-value2", "key2": "value2"}, false},
		} {
			if matcher.Matches(context.Background(), protocol.CommonHeader(c.requestHeader)) != c.matched {
				t.Errorf("No. %d case test failed", idx)
			}
		}
	})
	t.Run("regex header macth", func(t *testing.T) {
		headersConfig := []v2.HeaderMatcher{
			{
				Name:  "regexkey",
				Value: ".*",
				Regex: true,
			},
		}
		matcher := CreateHTTPHeaderMatcher(headersConfig)
		if !matcher.Matches(context.Background(), protocol.CommonHeader(map[string]string{"regexkey": "any"})) {
			t.Errorf("regex header match failed")
		}
	})
	t.Run("invalid regex header config", func(t *testing.T) {
		headersConfig := []v2.HeaderMatcher{
			{
				Name:  "regexkey",
				Value: "a)", // invalid regexp
				Regex: true,
			},
		}
		matcher := CreateHTTPHeaderMatcher(headersConfig)
		mimpl := matcher.(*httpHeaderMatcherImpl)
		if len(mimpl.headers) != 0 {
			t.Errorf("invalid regexkey should be ignored")
		}
	})
	t.Run("http method test", func(t *testing.T) {
		headersConfig := []v2.HeaderMatcher{
			{
				Name:  "method",
				Value: "POST",
			},
			{
				Name:  "common-key",
				Value: "common-value",
			},
		}
		ctx := variable.NewVariableContext(context.Background())
		variable.SetString(ctx, types.VarMethod, "POST")
		matcher := CreateHTTPHeaderMatcher(headersConfig)
		for idx, c := range []struct {
			requestHeader map[string]string
			ctx           context.Context
			matched       bool
		}{
			{
				// method in request header will be ignored.
				requestHeader: map[string]string{"common-key": "common-value", "method": "POST"},
				ctx:           context.Background(),
				matched:       false,
			},
			{
				requestHeader: map[string]string{"common-key": "common-value"},
				// method should be setted in the variables by the protocol stream modules
				ctx:     ctx,
				matched: true,
			},
			{
				requestHeader: map[string]string{"method": "POST"},
				ctx:           ctx, // only method matched, but headers not
				matched:       false,
			},
		} {
			if matcher.Matches(c.ctx, protocol.CommonHeader(c.requestHeader)) != c.matched {
				t.Errorf("No. %d case test failed", idx)
			}
		}
	})
}

// TODO: support query match in routers
func TestMatchQueryParams(t *testing.T) {
	qpm := queryParameterMatcherImpl{}
	configs := []v2.HeaderMatcher{
		{
			Name:  "key",
			Value: "value",
		},
		{
			Name:  "regex",
			Value: "[0-9]+",
			Regex: true,
		},
	}
	for _, c := range configs {
		if kv, err := NewKeyValueData(c); err == nil {
			qpm = append(qpm, kv)
		}
	}
	for idx, querys := range []struct {
		params   types.QueryParams
		expected bool
	}{
		{
			params: types.QueryParams(map[string]string{
				"key":   "value",
				"regex": "12345",
				"empty": "any",
			}),
			expected: true,
		},
		{
			params: types.QueryParams(map[string]string{
				"key":    "value",
				"regex":  "12345",
				"empty":  "",
				"ignore": "key",
			}),
			expected: true,
		},
		{
			params: types.QueryParams(map[string]string{
				"key": "value",
			}),
			expected: false,
		},
		{
			params: types.QueryParams(map[string]string{
				"key":   "value",
				"regex": "abc",
			}),
			expected: false,
		},
	} {
		if qpm.Matches(context.Background(), querys.params) != querys.expected {
			t.Fatalf("%d matched failed", idx)
		}
	}
}
