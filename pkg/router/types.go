package router

import (
	"strings"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/flowcontrol/ratelimit"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type Matchable interface {
	Match(headers map[string]string, randomValue uint64) types.Route
}

type RouterInfo interface {
	GetRouterName() string
}

type RouteBase interface {
	types.Route
	types.RouteRule
	Matchable
	RouterInfo
}

type ShadowPolicyImpl struct {
	cluster    string
	runtimeKey string
}

func (spi *ShadowPolicyImpl) ClusterName() string {
	return spi.cluster
}

func (spi *ShadowPolicyImpl) RuntimeKey() string {
	return spi.runtimeKey
}

type LowerCaseString struct {
	string_ string
}

func (lcs *LowerCaseString) Lower() {
	lcs.string_ = strings.ToLower(lcs.string_)
}

func (lcs *LowerCaseString) Equal(rhs types.LowerCaseString) bool {
	return lcs.string_ == rhs.Get()
}

func (lcs *LowerCaseString) Get() string {
	return lcs.string_
}

type HashPolicyImpl struct {
	hashImpl []*HashMethod
}

type HashMethod struct {
}

type DecoratorImpl struct {
	Operation string
}

func (di *DecoratorImpl) apply(span types.Span) {
	if di.Operation != "" {
		span.SetOperation(di.Operation)
	}
}

func (di *DecoratorImpl) getOperation() string {
	return di.Operation
}

type RateLimitPolicyImpl struct {
	rateLimitEntries []types.RateLimitPolicyEntry
	maxStageNumber   uint64
}

func (rp *RateLimitPolicyImpl) Enabled() bool {

	return true
}

func (rp *RateLimitPolicyImpl) GetApplicableRateLimit(stage string) []types.RateLimitPolicyEntry {

	return rp.rateLimitEntries
}

type RetryPolicyImpl struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
}

func (p *RetryPolicyImpl) RetryOn() bool {
	return p.retryOn
}

func (p *RetryPolicyImpl) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *RetryPolicyImpl) NumRetries() uint32 {
	return p.numRetries
}

type Loader struct{}

type MetadataMatchCriteriaImpl struct {
	metadataMatchCriteria []*types.MetadataMatchCriterion
}

func (mmcti *MetadataMatchCriteriaImpl) MetadataMatchCriteria() []*types.MetadataMatchCriterion {
	return mmcti.metadataMatchCriteria
}

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

// todo implement CorsPolicy

type RuntimeData struct {
	key          string
	defaultvalue uint64
}

type RateLimitPolicyEntryImpl struct {
	stage       uint64
	disablleKey string
	actions     RateLimitAction
}

func (rpei *RateLimitPolicyEntryImpl) Stage() uint64 {
	return rpei.stage
}

func (repi *RateLimitPolicyEntryImpl) DisableKey() string {
	return repi.disablleKey
}

func (repi *RateLimitPolicyEntryImpl) PopulateDescriptors(route types.RouteRule, descriptors []ratelimit.Descriptor, localSrvCluster string,
	headers map[string]string, remoteAddr string) {
}

type RateLimitAction interface{}

type WeightedClusterEntry struct {
	runtimeKey                   string
	loader                       Loader
	clusterWeight                uint64
	clusterMetadataMatchCriteria *MetadataMatchCriteriaImpl
}

type routerPolicy struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
}

func (p *routerPolicy) RetryOn() bool {
	return p.retryOn
}

func (p *routerPolicy) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *routerPolicy) NumRetries() uint32 {
	return p.numRetries
}

func (p *routerPolicy) RetryPolicy() types.RetryPolicy {
	return p
}

func (p *routerPolicy) ShadowPolicy() types.ShadowPolicy {
	return nil
}

func (p *routerPolicy) CorsPolicy() types.CorsPolicy {
	return nil
}

func (p *routerPolicy) LoadBalancerPolicy() types.LoadBalancerPolicy {
	return nil
}
