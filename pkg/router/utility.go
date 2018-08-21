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
	for key,value := range metadata {
		metadataMap[key] = types.GenerateHashedValue(value)
	}
	
	return metadataMap
}

// Note
// "runtimeKey" and "loader" are not used currently
func getWeightedClusterEntryAndVeirfy(totalClusterWeight uint32,weightedClusters []v2.WeightedCluster) (bool,
	map[string]weightedClusterEntry){
	var weightedClusterEntries = make(map[string]weightedClusterEntry)
	var totalWeight uint32 = 0
	
	for _,weightedCluster := range weightedClusters{
		totalWeight = totalWeight + weightedCluster.Cluster.Weight
		subsetLBMetaData := weightedCluster.Cluster.MetadataMatch
		
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