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
	"container/list"
	"regexp"

	"sort"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var ConfigUtilityInst = &ConfigUtility{}

type ConfigUtility struct {
	types.HeaderData
	QueryParameterMatcher
}

// types.MatchHeaders
func (cu *ConfigUtility) MatchHeaders(requestHeaders map[string]string, configHeaders []*types.HeaderData) bool {

	// step 1: match name
	// step 2: match value, if regex true, match pattern
	for _, cfgHeaderData := range configHeaders {
		cfgName := cfgHeaderData.Name.Get()
		cfgValue := cfgHeaderData.Value

		if value, ok := requestHeaders[cfgName]; ok {

			if !cfgHeaderData.IsRegex {
				if cfgValue != value {
					return false
				}
			} else {
				if !cfgHeaderData.RegexPattern.MatchString(value) {
					return false
				}
			}
		}
	}

	return true
}

// types.MatchQueryParams
func (cu *ConfigUtility) MatchQueryParams(queryParams *types.QueryParams, configQueryParams []types.QueryParameterMatcher) bool {

	for _, configQueryParam := range configQueryParams {

		if !configQueryParam.Matches(*queryParams) {
			return false
		}
	}

	return true
}

type QueryParameterMatcher struct {
	name         string
	value        string
	isRegex      bool
	regexPattern regexp.Regexp
}

func (qpm *QueryParameterMatcher) Matches(requestQueryParams types.QueryParams) bool {

	if requestQueryValue, ok := requestQueryParams[qpm.name]; !ok {
		return false
	} else if qpm.isRegex {
		return qpm.regexPattern.MatchString(requestQueryValue)
	} else if qpm.value == "" {
		return true
	} else {
		return qpm.value == requestQueryValue
	}

	return true
}

// Implementation of Config that reads from a proto file.
type ConfigImpl struct {
	name                  string
	routeMatcher          RouteMatcher
	internalOnlyHeaders   *list.List
	requestHeadersParser  *HeaderParser
	responseHeadersParser *HeaderParser
}

func (ci *ConfigImpl) Name() string {
	return ci.name
}

func (ci *ConfigImpl) Route(headers map[string]string, randomValue uint64) types.Route {
	return ci.routeMatcher.Route(headers, randomValue)
}

func (ci *ConfigImpl) InternalOnlyHeaders() *list.List {
	return ci.internalOnlyHeaders
}

//
func NewMetadataMatchCriteriaImpl(metadataMatches map[string]interface{}) *MetadataMatchCriteriaImpl {

	metadataMatchCriteriaImpl := &MetadataMatchCriteriaImpl{}
	metadataMatchCriteriaImpl.extractMetadataMatchCriteria(nil, metadataMatches)

	return metadataMatchCriteriaImpl
}

// realize sort.Sort
func (mmcti *MetadataMatchCriteriaImpl) Len() int {
	return len(mmcti.metadataMatchCriteria)
}

func (mmcti *MetadataMatchCriteriaImpl) Less(i, j int) bool {
	return mmcti.metadataMatchCriteria[i].MetadataKeyName() < mmcti.metadataMatchCriteria[j].MetadataKeyName()
}

func (mmcti *MetadataMatchCriteriaImpl) Swap(i, j int) {
	mmcti.metadataMatchCriteria[i], mmcti.metadataMatchCriteria[j] = mmcti.metadataMatchCriteria[j],
		mmcti.metadataMatchCriteria[i]
}

type MetadataMatchCriteriaImpl struct {
	metadataMatchCriteria []types.MetadataMatchCriterion
}

func (mmcti *MetadataMatchCriteriaImpl) MetadataMatchCriteria() []types.MetadataMatchCriterion {
	return mmcti.metadataMatchCriteria
}

func (mmcti *MetadataMatchCriteriaImpl) MergeMatchCriteria(metadataMatches map[string]interface{}) types.MetadataMatchCriteria {
	return nil
}

func (mmcti *MetadataMatchCriteriaImpl) metadataMatchCriteriaImpl(criteria []types.MetadataMatchCriterion) {
	mmcti.metadataMatchCriteria = criteria
}

// used to generate metadata match criteria from config
func (mmcti *MetadataMatchCriteriaImpl) extractMetadataMatchCriteria(parent *MetadataMatchCriteriaImpl,
	metadataMatches map[string]interface{}) {

	var mdMatchCriteria []types.MetadataMatchCriterion

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

		if vs, ok := v.(string); ok {
			mmci := &MetadataMatchCriterionImpl{
				name:  k,
				value: types.GenerateHashedValue(vs),
			}

			if index, ok := existingMap[k]; ok {

				// update value
				mdMatchCriteria[index] = mmci
			} else {
				// append
				mdMatchCriteria = append(mdMatchCriteria, mmci)
			}

		} else {
			log.DefaultLogger.Errorf("Currently,metadata only support map[string]string type")
		}
	}

	mmcti.metadataMatchCriteria = mdMatchCriteria

	// sorting in lexically by name
	sort.Sort(mmcti)
}

//
type MetadataMatchCriterionImpl struct {
	name  string
	value types.HashedValue
}

func (mmci *MetadataMatchCriterionImpl) MetadataKeyName() string {
	return mmci.name
}

func (mmci *MetadataMatchCriterionImpl) Value() types.HashedValue {
	return mmci.value
}
