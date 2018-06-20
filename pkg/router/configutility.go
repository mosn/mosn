package router

import (
	"regexp"
	"strings"
	"gitlab.alipay-inc.com/afe/mosn/pkg/flowcontrol/ratelimit"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type LowerCaseString struct {
	string_ string
}

func (lcs *LowerCaseString) lower() {
	lcs.string_ = strings.ToLower(lcs.string_)
}

func (lcs *LowerCaseString) equal(rhs *LowerCaseString) bool {
	
	return lcs.string_ == rhs.string_
}

func (lcs *LowerCaseString) get() string {
	return lcs.string_
}

type HeaderData struct {
	name LowerCaseString
	value string
	isRegex bool
	regexPattern regexp.Regexp
	
}

type QueryParameterMatcher struct {
	name string
	value string
	isRegex bool
	regexPattern regexp.Regexp
}

func (qpm *QueryParameterMatcher) matches (requestQueryParams *QueryParams) bool {

	return false
}



type QueryParams map[string]string


type RateLimitPolicyEntryImpl struct {
	stage uint64
	disablleKey  string
	actions RateLimitAction
}

func (rpei *RateLimitPolicyEntryImpl) Stage() uint64{
	return rpei.stage
}

func (repi *RateLimitPolicyEntryImpl)DisableKey() string{
	return  repi.disablleKey
}

func (repi *RateLimitPolicyEntryImpl) PopulateDescriptors(route types.RouteRule, descriptors []ratelimit.Descriptor, localSrvCluster string,
	headers map[string]string, remoteAddr string){
}

type RateLimitAction interface{}

type ShadowPolicyImpl struct {
	cluster string
	runtimeKey string
}

func (spi *ShadowPolicyImpl) ClusterName() string {
	return spi.cluster
}

func (spi *ShadowPolicyImpl) RuntimeKey() string {
	return spi.runtimeKey
}