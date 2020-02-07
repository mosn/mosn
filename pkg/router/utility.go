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
	"regexp"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func getWeightedClusterEntry(weightedClusters []v2.WeightedCluster) (map[string]weightedClusterEntry, uint32) {
	weightedClusterEntries := make(map[string]weightedClusterEntry)
	var totalWeight uint32
	for _, weightedCluster := range weightedClusters {
		weightedClusterEntries[weightedCluster.Cluster.Name] = weightedClusterEntry{
			clusterName:                  weightedCluster.Cluster.Name,
			clusterWeight:                weightedCluster.Cluster.Weight,
			clusterMetadataMatchCriteria: NewMetadataMatchCriteriaImpl(weightedCluster.Cluster.MetadataMatch),
		}
		totalWeight = totalWeight + weightedCluster.Cluster.Weight
	}
	return weightedClusterEntries, totalWeight
}

// GetRouterHeaders exports getRouterHeaders
func GetRouterHeaders(headers []v2.HeaderMatcher) []*types.HeaderData {
	return getRouterHeaders(headers)
}

func getRouterHeaders(headers []v2.HeaderMatcher) []*types.HeaderData {
	var headerDatas []*types.HeaderData

	for _, header := range headers {
		headerData := &types.HeaderData{
			Name: &lowerCaseString{
				header.Name,
			},
			Value:   header.Value,
			IsRegex: header.Regex,
		}

		if header.Regex {
			pattern, err := regexp.Compile(header.Value)
			if err != nil {
				log.DefaultLogger.Errorf("getRouterHeaders compile error")
				continue
			}
			headerData.RegexPattern = pattern
		}

		headerDatas = append(headerDatas, headerData)

	}

	return headerDatas
}

func getHeaderParser(headersToAdd []*v2.HeaderValueOption, headersToRemove []string) *headerParser {
	if headersToAdd == nil && headersToRemove == nil {
		return nil
	}

	return &headerParser{
		headersToAdd:    getHeaderPair(headersToAdd),
		headersToRemove: getHeadersToRemove(headersToRemove),
	}
}

func getHeaderPair(headersToAdd []*v2.HeaderValueOption) []*headerPair {
	if headersToAdd == nil {
		return nil
	}
	headerPairs := make([]*headerPair, 0, len(headersToAdd))
	for _, option := range headersToAdd {
		key := &lowerCaseString{
			option.Header.Key,
		}
		key.Lower()

		// set true to Append as default
		isAppend := true
		if option.Append != nil {
			isAppend = *option.Append
		}
		value := getHeaderFormatter(option.Header.Value, isAppend)
		if value == nil {
			continue
		}
		headerPairs = append(headerPairs, &headerPair{
			headerName:      key,
			headerFormatter: value,
		})
	}
	return headerPairs
}

func getHeadersToRemove(headersToRemove []string) []*lowerCaseString {
	if headersToRemove == nil {
		return nil
	}
	lowerCaseHeaders := make([]*lowerCaseString, 0, len(headersToRemove))
	for _, header := range headersToRemove {
		lowerCaseHeader := &lowerCaseString{
			str: header,
		}
		lowerCaseHeader.Lower()
		lowerCaseHeaders = append(lowerCaseHeaders, lowerCaseHeader)
	}
	return lowerCaseHeaders
}
