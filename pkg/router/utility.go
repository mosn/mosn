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
	"github.com/alipay/sofa-mosn/pkg/log"
)

// getClusterMosnLBMetaDataMap from v2.Metadata
// e.g. metadata =  { "filter_metadata": {"mosn.lb": { "label": "gray"  } } }
// 4-tier map
func getClusterMosnLBMetaDataMap(metadata v2.Metadata) types.RouteMetaData {
	metadataMap := make(map[string]types.HashedValue)
	
	if metadataInterface, ok := metadata[types.RouterMetadataKey]; ok {
		if value, ok := metadataInterface.(map[string]interface{}); ok {
			if mosnLbInterface, ok := value[types.RouterMetadataKeyLb]; ok {
				if mosnLb, ok := mosnLbInterface.(map[string]interface{}); ok {
					for k, v := range mosnLb {
						if vs, ok := v.(string); ok {
							metadataMap[k] = types.GenerateHashedValue(vs)
						} else {
							log.DefaultLogger.Fatal("Currently,only map[string]string type is supported for metadata")
						}
					}
				}
			}
		}
	}
	
	return metadataMap
}

// getMosnLBMetaData
// get mosn lb metadata from config
func getMosnLBMetaData(metadata v2.Metadata) map[string]interface{} {
	if metadataInterface, ok := metadata[types.RouterMetadataKey]; ok {
		if value, ok := metadataInterface.(map[string]interface{}); ok {
			if mosnLbInterface, ok := value[types.RouterMetadataKeyLb]; ok {
				if mosnLb, ok := mosnLbInterface.(map[string]interface{}); ok {
					return mosnLb
				}
			}
		}
	}
	
	return nil
}

// Note
// "runtimeKey" and "loader" are not used currently
func getWeightedClusterEntryAndVeirfy(totalClusterWeight uint32,weightedClusters []v2.WeightedCluster) (bool,
	map[string]weightedClusterEntry){
	var weightedClusterEntries = make(map[string]weightedClusterEntry)
	var totalWeight uint32 = 0
	
	for _,weightedCluster := range weightedClusters{
		totalWeight = totalWeight + weightedCluster.Cluster.Weight
		subsetLBMetaData := getMosnLBMetaData(weightedCluster.Cluster.MetadataMatch)
		
		weightedClusterEntries[weightedCluster.Cluster.Name] = weightedClusterEntry{
			clusterName:weightedCluster.Cluster.Name,
			clusterWeight:weightedCluster.Cluster.Weight,
			clusterMetadataMatchCriteria:NewMetadataMatchCriteriaImpl(subsetLBMetaData),
			
		}
	}
	
	if totalWeight == totalClusterWeight {
		return true,weightedClusterEntries
	}
	
	return false,nil
	
}
