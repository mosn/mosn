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
	"reflect"
	"sort"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
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
				MatchCriteriaArray: []types.MetadataMatchCriterion{
					&MetadataMatchCriterionImpl{
						Name:  "appInfo",
						Value: types.GenerateHashedValue("test"),
					},
					&MetadataMatchCriterionImpl{
						Name:  "label",
						Value: types.GenerateHashedValue("green"),
					},
					&MetadataMatchCriterionImpl{
						Name:  "version",
						Value: types.GenerateHashedValue("v1"),
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
				MatchCriteriaArray: []types.MetadataMatchCriterion{
					&MetadataMatchCriterionImpl{
						Name:  "appInfo",
						Value: types.GenerateHashedValue("test"),
					},
					&MetadataMatchCriterionImpl{
						Name:  "label",
						Value: types.GenerateHashedValue("green"),
					},
					&MetadataMatchCriterionImpl{
						Name:  "version",
						Value: types.GenerateHashedValue("v1"),
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
				MatchCriteriaArray: []types.MetadataMatchCriterion{
					&MetadataMatchCriterionImpl{
						Name:  "version",
						Value: types.GenerateHashedValue("v1"),
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
					ResponseHeadersToRemove: []string{"status"},
				},
			},
			want: &configImpl{
				requestHeadersParser: &headerParser{
					headersToAdd: []*headerPair{
						{
							headerName: &lowerCaseString{"level"},
							headerFormatter: &plainHeaderFormatter{
								isAppend:    false,
								staticValue: "1",
							},
						},
					},
				},
				responseHeadersParser: &headerParser{
					headersToAdd: []*headerPair{
						{
							headerName: &lowerCaseString{"random"},
							headerFormatter: &plainHeaderFormatter{
								isAppend:    false,
								staticValue: "123456",
							},
						},
					},
					headersToRemove: []*lowerCaseString{{"status"}},
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
	var mciArray []types.MetadataMatchCriterion
	for i := range keys {
		mmci := &MetadataMatchCriterionImpl{
			Name:  keys[i],
			Value: types.GenerateHashedValue(values[i]),
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
