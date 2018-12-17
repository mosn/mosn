package sink

import (
	"fmt"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// MetricsSinkCreator creates a MetricsSink according to config
type MetricsSinkCreator func(config map[string]interface{}) (types.MetricsSink, error)

var metricsSinkFactory map[string]MetricsSinkCreator

func init() {
	metricsSinkFactory = make(map[string]MetricsSinkCreator)
}

// RegisterSink registers the sinkType as MetricsSinkCreator
func RegisterSink(sinkType string, creator MetricsSinkCreator) {
	metricsSinkFactory[sinkType] = creator
}

// CreateStreamFilterChainFactory creates a StreamFilterChainFactory according to filterType
func CreateMetricsSink(sinkType string, config map[string]interface{}) (types.MetricsSink, error) {
	if creator, ok := metricsSinkFactory[sinkType]; ok {
		sink, err := creator(config)
		if err != nil {
			return nil, fmt.Errorf("create metrics sink failed: %v", err)
		}
		return sink, nil
	}
	return nil, fmt.Errorf("unsupported metrics sink type: %v", sinkType)
}
