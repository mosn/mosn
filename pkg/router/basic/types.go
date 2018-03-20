package basic

import "gitlab.alipay-inc.com/afe/mosn/pkg"
import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

type RouteRuleImplAdaptor struct {
}

func (r *RouteRuleImplAdaptor) ClusterName() string {
	return ""
}

func (r *RouteRuleImplAdaptor) Priority() pkg.Priority {
	return pkg.NORMAL
}

func (r *RouteRuleImplAdaptor) VirtualCluster(headers map[string]string) types.VirtualCluster {
	return nil
}

func (r *RouteRuleImplAdaptor) Policy() types.Policy {
	return nil
}

func (r *RouteRuleImplAdaptor) MetadataMatcher() types.MetadataMatcher {
	return nil
}
