package filter

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

type StreamFilterFactoryCreator func(config map[string]interface{}) (types.StreamFilterChainFactory, error)
