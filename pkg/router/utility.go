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