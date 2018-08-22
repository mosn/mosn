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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

func Test_getWeightedClusterEntryAndVerify(t *testing.T) {
	type args struct {
		totalClusterWeight uint32
		weightedClusters   []v2.WeightedCluster
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
				totalClusterWeight: 100,
				weightedClusters: []v2.WeightedCluster{
					{Cluster: v2.ClusterWeight{Name: "c1", Weight: 50, MetadataMatch: v2.Metadata{"label": "green", "version": "v1"}}},
					{Cluster: v2.ClusterWeight{Name: "c2", Weight: 30, MetadataMatch: v2.Metadata{"label": "blue", "version": "v2"}}},
					{Cluster: v2.ClusterWeight{Name: "c3", Weight: 20, MetadataMatch: v2.Metadata{"label": "gray", "version": "v0"}}},
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
		{
			name: "case1",
			args: args{
				totalClusterWeight: 100,
				weightedClusters: []v2.WeightedCluster{
					{Cluster: v2.ClusterWeight{Name: "c1", Weight: 50, MetadataMatch: v2.Metadata{"label": "green", "version": "v1"}}},
					{Cluster: v2.ClusterWeight{Name: "c2", Weight: 30, MetadataMatch: v2.Metadata{"label": "blue", "version": "v2"}}},
					{Cluster: v2.ClusterWeight{Name: "c3", Weight: 10, MetadataMatch: v2.Metadata{"label": "gray", "version": "v0"}}},
				},
			},
			want: result{
				valid: false,
				value: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := getWeightedClusterEntryAndVerify(tt.args.totalClusterWeight, tt.args.weightedClusters)
			if ok != tt.want.valid {
				t.Errorf("get weighted cluster entry and verify name = %s got = %v, want %v", tt.name, ok, tt.want.valid)
			}
			if !reflect.DeepEqual(entry, tt.want.value) {
				t.Errorf("get weighted cluster entry and verify name = %s got1 = %v, want %v", tt.want, entry, tt.want.value)
			}
		})
	}
}
