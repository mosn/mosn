package cluster

import (
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type subSetLoadBalancer struct {
	lbType         types.LoadBalancerType // LB for choosing subset's  host
	fallBackPolicy types.FallBackPolicy
	defaultSubSet  types.HostSet
	subSetKeys     [][]string //
}

func NewSubsetLoadBalancer(lbType types.LoadBalancerType, ssbc v2.LBSubsetConfig) types.SubSetLoadBalancer {
	return &subSetLoadBalancer{
		lbType:         lbType,
		fallBackPolicy: types.FallBackPolicy(ssbc.FallBackPolicy),
		// todo types.Host <-> // LB for choosing subset's  host
		defaultSubSet: &hostSet{},
		subSetKeys:    ssbc.SubsetSelectors,
	}
}

func (sslb *subSetLoadBalancer) TryChooseHostFromContext(context *context.Context, hostChosen *bool) {

}

func (sslb *subSetLoadBalancer) HostMatchesDefaultSubset(host types.Host) bool {
	return true
}

func (sslb *subSetLoadBalancer) HostMatches(kvs *types.SubsetMetadata, host *types.Host) bool {

	return true
}

func (sslb *subSetLoadBalancer) FindSubset(matches []types.MetadataMatchEntry) *types.LBSubsetEntry {
	return nil
}

func (sslb *subSetLoadBalancer) FindOrCreateSubset(subsets *types.LbSubsetMap, kvs *types.SubsetMetadata, idx uint32) {

}

func (sslb *subSetLoadBalancer) ExtractSubsetMetadata(subsetKeys []string, host types.Host) {

}
