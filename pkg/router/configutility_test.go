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
	"testing"

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
