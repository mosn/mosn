package router

import (
	"container/list"
	"regexp"

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
