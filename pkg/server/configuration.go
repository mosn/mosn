package server

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

func buildFilterChain(filterManager types.FilterManager, factory types.NetworkFilterFactoryCb) bool {
	factory(filterManager)

	return filterManager.InitializeReadFilters()
}
