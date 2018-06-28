package basic

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

type RouteRuleImplAdaptor struct {
}

func (r *RouteRuleImplAdaptor) ClusterName() string {
	return ""
}

func (r *RouteRuleImplAdaptor) Priority() types.Priority {
	return types.PriorityDefault
}

func (r *RouteRuleImplAdaptor) VirtualCluster(headers map[string]string) types.VirtualCluster {
	return nil
}

func (r *RouteRuleImplAdaptor) Policy() types.Policy {
	return nil
}

//func (r *RouteRuleImplAdaptor) MetadataMatcher() types.MetadataMatcher {
//	return nil
//}

func (r *RouteRuleImplAdaptor) VirtualHost () types.VirtualHost{
	return nil
}

func (r *RouteRuleImplAdaptor) Metadata() types.RouteMetaData {
	return nil
}

func (r *RouteRuleImplAdaptor) MetadataMatchCriteria() types.MetadataMatchCriteria {
	return nil
}