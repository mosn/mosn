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

package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
	"mosn.io/mosn/pkg/types"
)

func TestSubsetLbBuilder_InitIndex(t *testing.T) {
	subsetInfo := NewLBSubsetInfo(exampleSubsetConfig())
	hosts := exampleHostConfigs()
	builder := newSubsetLoadBalancerBuilder(&clusterInfo{lbSubsetInfo: subsetInfo, stats: newClusterStats("test")}, createHostset(hosts))
	merged := make(map[string]map[string][]int)
	for i, host := range hosts {
		for _, key := range subsetMergeKeys(subsetInfo.SubsetKeys(), subsetInfo.DefaultSubset()) {
			val, ok := host.MetaData[key]
			if !ok {
				continue
			}
			if _, ok := merged[key]; !ok {
				merged[key] = make(map[string][]int)
			}
			merged[key][val] = append(merged[key][val], i)
		}
	}
	require.Len(t, builder.indexer, len(merged))
	for key, vals := range merged {
		require.Len(t, builder.indexer[key], len(vals))
		for val, hosts := range vals {
			indexArray := builder.indexer[key][val].AppendTo(nil)
			require.Equal(t, hosts, indexArray)
		}
	}
}

func TestSubsetLbBuilder_MetadataCombinations(t *testing.T) {
	subsetInfo := NewLBSubsetInfo(exampleSubsetConfig())
	builder := newSubsetLoadBalancerBuilder(&clusterInfo{lbSubsetInfo: subsetInfo}, createHostset(exampleHostConfigs()))
	var kvList []types.SubsetMetadata
	kvList = builder.metadataCombinations([]string{"version"})
	// [[{version 1.0}] [{version 1.1}] [{version 1.2-pre}]]
	require.Len(t, kvList, 3)
	for _, kvs := range kvList {
		require.Len(t, kvs, 1)
	}
	kvList = builder.metadataCombinations([]string{"stage", "type"})
	// [[{stage prod} {type bigmem}] [{stage prod} {type std}] [{stage dev} {type std}] [{stage dev} {type bigmem}]]
	require.Len(t, kvList, 4)
	for _, kvs := range kvList {
		require.Len(t, kvs, 2)
	}

	kvList = builder.metadataCombinations([]string{"stage", "version"})
	// [[{stage prod} {version 1.0}] [{stage prod} {version 1.1}] [{stage prod} {version 1.2-pre}] [{stage dev} {version 1.0}] [{stage dev} {version 1.1}] [{stage dev} {version 1.2-pre}]]
	require.Len(t, kvList, 6)
	for _, kvs := range kvList {
		require.Len(t, kvs, 2)
	}
}

func TestSubsetMergeKeys(t *testing.T) {
	keys := subsetMergeKeys([]types.SortedStringSetType{
		types.InitSet([]string{"1", "2", "3"}),
		types.InitSet([]string{"2", "3", "4"}),
	}, []types.Pair{
		{
			T1: "5",
			T2: "6",
		},
	})
	require.Len(t, keys, 5)
}
