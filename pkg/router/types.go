package router

import (
	"strings"
	"time"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/flowcontrol/ratelimit"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type  HeaderParser struct {
	headersToAdd    []types.Pair
	headersToRemove []*LowerCaseString
}

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
	loader                       types.Loader
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

// e.g. metadata =  { "filter_metadata": {"envoy.lb": { "label": "gray"  } } }
// 4-tier map
func GetClusterEnvoyLBMetaDataMap(metadata v2.Metadata) types.RouteMetaData {
	metadataMap := make(map[string]types.HashedValue)

	if metadataInterface, ok := metadata[types.RouterMatadataKey]; ok {
		if value, ok := metadataInterface.(map[string]interface{}); ok {
			if envoyLbInterface, ok := value[types.RouterMetadataKeyLb]; ok {
				if envoyLb, ok := envoyLbInterface.(map[string]interface{}); ok {
					for k, v := range envoyLb {
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

// get envoy lb metadata from config
func GetEnvoyLBMetaData(route *v2.Router) map[string]interface{} {
	if metadataInterface, ok := route.Route.MetadataMatch[types.RouterMatadataKey]; ok {
		if value, ok := metadataInterface.(map[string]interface{}); ok {
			if envoyLbInterface, ok := value[types.RouterMetadataKeyLb]; ok {
				if envoyLb, ok := envoyLbInterface.(map[string]interface{}); ok {
					return envoyLb
				}
			}
		}
	}

	return nil
}
