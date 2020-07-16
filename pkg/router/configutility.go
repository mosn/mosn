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
	"sort"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

var ConfigUtilityInst = &configUtility{}

type configUtility struct {
	types.HeaderData
	queryParameterMatcher
}

// types.MatchHeaders
func (cu *configUtility) MatchHeaders(requestHeaders api.HeaderMap, configHeaders []*types.HeaderData) bool {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "config utility", "try match header", requestHeaders)
	}
	for _, cfgHeaderData := range configHeaders {
		cfgName := cfgHeaderData.Name.Get()
		cfgValue := cfgHeaderData.Value
		if cfgName == "method" {
			cfgName = protocol.MosnHeaderMethod
		}

		// if a condition is not matched, return false
		// all condition matched, return true
		value, ok := requestHeaders.Get(cfgName)
		if !ok {
			return false
		}
		if cfgHeaderData.IsRegex {
			if !cfgHeaderData.RegexPattern.MatchString(value) {
				return false
			}
		} else {
			if cfgValue != value {
				return false
			}
		}
	}
	return true
}

// types.MatchQueryParams
func (cu *configUtility) MatchQueryParams(queryParams types.QueryParams, configQueryParams []types.QueryParameterMatcher) bool {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf(RouterLogFormat, "config utility", "try match query params", queryParams)
	}
	// if a condition is not matched, return false
	// all condition matched, return true
	for _, configQueryParam := range configQueryParams {
		if !configQueryParam.Matches(queryParams) {
			return false
		}
	}
	return true
}

type queryParameterMatcher struct {
	name         string
	value        string
	isRegex      bool
	regexPattern regexp.Regexp
}

func (qpm *queryParameterMatcher) Matches(requestQueryParams types.QueryParams) bool {
	requestQueryValue, ok := requestQueryParams[qpm.name]
	if !ok {
		return false
	}
	if qpm.isRegex {
		return qpm.regexPattern.MatchString(requestQueryValue)
	}
	if qpm.value == "" {
		return true
	}
	return qpm.value == requestQueryValue
}

// NewConfigImpl return an configImpl instance contains requestHeadersParser and responseHeadersParser
func NewConfigImpl(routerConfig *v2.RouterConfiguration) *configImpl {
	return &configImpl{
		requestHeadersParser:  getHeaderParser(routerConfig.RequestHeadersToAdd, nil),
		responseHeadersParser: getHeaderParser(routerConfig.ResponseHeadersToAdd, routerConfig.ResponseHeadersToRemove),
	}
}

// Implementation of Config that reads from a proto file.
// TODO: more action
type configImpl struct {
	requestHeadersParser  *headerParser
	responseHeadersParser *headerParser
}

// NewMetadataMatchCriteriaImpl
func NewMetadataMatchCriteriaImpl(metadataMatches map[string]string) *MetadataMatchCriteriaImpl {

	metadataMatchCriteriaImpl := &MetadataMatchCriteriaImpl{}
	metadataMatchCriteriaImpl.extractMetadataMatchCriteria(nil, metadataMatches)

	return metadataMatchCriteriaImpl
}

// MetadataMatchCriteriaImpl class wrapper MatchCriteriaArray
// which contains MatchCriteria in dictionary sorted
type MetadataMatchCriteriaImpl struct {
	MatchCriteriaArray []api.MetadataMatchCriterion
}

// MetadataMatchCriteria
func (mmcti *MetadataMatchCriteriaImpl) MetadataMatchCriteria() []api.MetadataMatchCriterion {
	return mmcti.MatchCriteriaArray
}

// MergeMatchCriteria
// No usage currently
func (mmcti *MetadataMatchCriteriaImpl) MergeMatchCriteria(metadataMatches map[string]interface{}) api.MetadataMatchCriteria {
	return nil
}

func (mmcti *MetadataMatchCriteriaImpl) Len() int {
	return len(mmcti.MatchCriteriaArray)
}

func (mmcti *MetadataMatchCriteriaImpl) Less(i, j int) bool {
	return mmcti.MatchCriteriaArray[i].MetadataKeyName() < mmcti.MatchCriteriaArray[j].MetadataKeyName()
}

func (mmcti *MetadataMatchCriteriaImpl) Swap(i, j int) {
	mmcti.MatchCriteriaArray[i], mmcti.MatchCriteriaArray[j] = mmcti.MatchCriteriaArray[j],
		mmcti.MatchCriteriaArray[i]
}

// Used to generate metadata match criteria from config
func (mmcti *MetadataMatchCriteriaImpl) extractMetadataMatchCriteria(parent *MetadataMatchCriteriaImpl,
	metadataMatches map[string]string) {

	var mdMatchCriteria []api.MetadataMatchCriterion
	// used to record key and its index for o(1) searching
	var existingMap = make(map[string]uint32)

	// get from parent
	if nil != parent {
		for _, v := range parent.MetadataMatchCriteria() {
			existingMap[v.MetadataKeyName()] = uint32(len(mdMatchCriteria))
			mdMatchCriteria = append(mdMatchCriteria, v)
		}
	}

	// get from metadatamatch
	for k, v := range metadataMatches {
		mmci := &MetadataMatchCriterionImpl{
			Name:  k,
			Value: v,
		}

		if index, ok := existingMap[k]; ok {
			// update value
			mdMatchCriteria[index] = mmci
		} else {
			// append
			mdMatchCriteria = append(mdMatchCriteria, mmci)
		}
	}

	mmcti.MatchCriteriaArray = mdMatchCriteria
	// sorting in lexically by name
	sort.Sort(mmcti)
}

// MetadataMatchCriterionImpl class contains the name and value of the metadata match criterion
// Implement types.MetadataMatchCriterion
type MetadataMatchCriterionImpl struct {
	Name  string
	Value string
}

// MetadataKeyName return name
func (mmci *MetadataMatchCriterionImpl) MetadataKeyName() string {
	return mmci.Name
}

// MetadataValue return value
func (mmci *MetadataMatchCriterionImpl) MetadataValue() string {
	return mmci.Value
}
