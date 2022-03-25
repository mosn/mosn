package xds

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

type mockCMF struct{}

func (cmf *mockCMF) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {}

type mockNetworkFilterFactory struct{}

func (ff *mockNetworkFilterFactory) CreateFilterChain(context context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
}

func CreateMockFilerFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	return &mockNetworkFilterFactory{}, nil
}

func init() {
	api.RegisterNetwork("proxy", CreateMockFilerFactory)
}
