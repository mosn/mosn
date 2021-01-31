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

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func Test_getWeightedClusterEntryAndVerify(t *testing.T) {
	type args struct {
		weightedClusters []v2.WeightedCluster
	}

	type result struct {
		valid bool
		value map[string]weightedClusterEntry
	}

	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "case1",
			args: args{
				weightedClusters: []v2.WeightedCluster{
					{Cluster: v2.ClusterWeight{ClusterWeightConfig: v2.ClusterWeightConfig{Name: "c1", Weight: 50}, MetadataMatch: api.Metadata{"label": "green", "version": "v1"}}},
					{Cluster: v2.ClusterWeight{ClusterWeightConfig: v2.ClusterWeightConfig{Name: "c2", Weight: 30}, MetadataMatch: api.Metadata{"label": "blue", "version": "v2"}}},
					{Cluster: v2.ClusterWeight{ClusterWeightConfig: v2.ClusterWeightConfig{Name: "c3", Weight: 20}, MetadataMatch: api.Metadata{"label": "gray", "version": "v0"}}},
				},
			},
			want: result{
				valid: true,
				value: map[string]weightedClusterEntry{
					"c1": weightedClusterEntry{
						clusterName:                  "c1",
						clusterWeight:                50,
						clusterMetadataMatchCriteria: NewMetadataMatchCriteriaImpl(map[string]string{"label": "green", "version": "v1"}),
					},
					"c2": weightedClusterEntry{
						clusterName:                  "c2",
						clusterWeight:                30,
						clusterMetadataMatchCriteria: NewMetadataMatchCriteriaImpl(map[string]string{"label": "blue", "version": "v2"}),
					},
					"c3": weightedClusterEntry{
						clusterName:                  "c3",
						clusterWeight:                20,
						clusterMetadataMatchCriteria: NewMetadataMatchCriteriaImpl(map[string]string{"label": "gray", "version": "v0"}),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, _ := getWeightedClusterEntry(tt.args.weightedClusters)
			if !reflect.DeepEqual(entry, tt.want.value) {
				t.Errorf("get weighted cluster entry and verify name = %s got1 = %v, want %v", tt.name, entry, tt.want.value)
			}
		})
	}
}

func Test_getHeaderParser(t *testing.T) {

	type args struct {
		headersToAdd    []*v2.HeaderValueOption
		headersToRemove []string
	}

	FALSE := false

	tests := []struct {
		name string
		args args
		want *headerParser
	}{
		{
			name: "case1",
			args: args{
				headersToAdd: []*v2.HeaderValueOption{
					{
						Header: &v2.HeaderValue{
							Key:   "LEVEL",
							Value: "1",
						},
						Append: &FALSE,
					},
				},
				headersToRemove: []string{"STATUS"},
			},
			want: &headerParser{
				headersToAdd: []*headerPair{
					{
						headerName: "level",
						headerFormatter: &plainHeaderFormatter{
							isAppend:    false,
							staticValue: "1",
						},
					},
				},
				headersToRemove: []string{"status"},
			},
		},
		{
			name: "case2",
			args: args{
				headersToAdd:    nil,
				headersToRemove: nil,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHeaderParser(tt.args.headersToAdd, tt.args.headersToRemove); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHeaderParser(headersToAdd []*v2.HeaderValueOption, headersToRemove []string) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getHeaderPair(t *testing.T) {

	type args struct {
		headersToAdd []*v2.HeaderValueOption
	}

	FALSE := false

	tests := []struct {
		name string
		args args
		want []*headerPair
	}{
		{
			name: "case1",
			args: args{
				headersToAdd: []*v2.HeaderValueOption{
					{
						Header: &v2.HeaderValue{
							Key:   "LEVEL",
							Value: "1",
						},
						Append: &FALSE,
					},
				},
			},
			want: []*headerPair{
				{
					headerName: "level",
					headerFormatter: &plainHeaderFormatter{
						isAppend:    false,
						staticValue: "1",
					},
				},
			},
		},
		{
			name: "case2",
			args: args{
				headersToAdd: nil,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHeaderPair(tt.args.headersToAdd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHeaderPair(headersToAdd []*v2.HeaderValueOption) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getHeadersToRemove(t *testing.T) {

	type args struct {
		headersToRemove []string
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "case1",
			args: args{
				headersToRemove: []string{"STATUS"},
			},
			want: []string{"status"},
		},
		{
			name: "case2",
			args: args{
				headersToRemove: nil,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHeadersToRemove(tt.args.headersToRemove); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHeadersToRemove(headersToRemove []string) = %v, want %v", got, tt.want)
			}
		})
	}
}
