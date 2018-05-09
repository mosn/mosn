package filter

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/faultinject"
	"gitlab.alipay-inc.com/afe/mosn/pkg/filter/stream/healthcheck/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var creatorFactory map[string]StreamFilterFactoryCreator

func init() {
	creatorFactory = make(map[string]StreamFilterFactoryCreator)
	//reg
	Register("fault_inject", faultinject.CreateFaultInjectFilterFactory)
	Register("healthcheck", sofarpc.CreateHealthCheckFilterFactory)
}

func Register(filterType string, creator StreamFilterFactoryCreator) {
	creatorFactory[filterType] = creator
}

func CreateStreamFilterChainFactory(filterType string, config map[string]interface{}) types.StreamFilterChainFactory {

	if cf, ok := creatorFactory[filterType]; ok {
		sfcf, err := cf(config)

		if err != nil {
			log.StartLogger.Fatalln("create stream filter chain factory failed: ", err)
		}

		return sfcf
	} else {
		log.StartLogger.Fatalln("unsupport stream filter type: ", filterType)
		return nil
	}
}
