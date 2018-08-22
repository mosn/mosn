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
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// getClusterMosnLBMetaDataMap from v2.Metadata
// e.g. metadata =  { "filter_metadata": {"mosn.lb": { "label": "gray"  } } }
// 4-tier map
func getClusterMosnLBMetaDataMap(metadata v2.Metadata) types.RouteMetaData {
	metadataMap := make(map[string]types.HashedValue)
	for key, value := range metadata {
		metadataMap[key] = types.GenerateHashedValue(value)
	}

	return metadataMap
}

// Note
// "runtimeKey" and "loader" are not used currently
func getWeightedClusterEntryAndVerify(totalClusterWeight uint32, weightedClusters []v2.WeightedCluster) (map[string]weightedClusterEntry, bool) {
	var weightedClusterEntries = make(map[string]weightedClusterEntry)
	var totalWeight uint32 = 0

	for _, weightedCluster := range weightedClusters {
		totalWeight = totalWeight + weightedCluster.Cluster.Weight
		subsetLBMetaData := weightedCluster.Cluster.MetadataMatch

		weightedClusterEntries[weightedCluster.Cluster.Name] = weightedClusterEntry{
			clusterName:                  weightedCluster.Cluster.Name,
			clusterWeight:                weightedCluster.Cluster.Weight,
			clusterMetadataMatchCriteria: NewMetadataMatchCriteriaImpl(subsetLBMetaData),
		}
	}

	if totalWeight == totalClusterWeight {
		return weightedClusterEntries, true
	}

	return nil, false
}
