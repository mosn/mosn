package sink

import (
	"github.com/alipay/sofa-mosn/pkg/config"
	"errors"
	"github.com/alipay/sofa-mosn/pkg/types"
	"time"
	"github.com/alipay/sofa-mosn/pkg/stats"
)

var sinks []types.MetricsSink
var defaultFlushInteval = time.Second

func init() {
	config.RegisterConfigParsedListener(config.ParseCallbackKeyMetrics, initMetricsSink)
}

func initMetricsSink(data interface{}, endParsing bool) error {
	if data == nil {
		return nil
	}

	mc, ok := data.(config.MetricsConfig)
	if !ok {
		return errors.New("unknown data while init metrics sink")
	}

	// create sinks
	for _, cfg := range mc.SinkConfigs {
		sink, err := CreateMetricsSink(cfg.Type, cfg.Config)
		// abort
		if err != nil {
			return err
		}
		sinks = append(sinks, sink)
	}

	if mc.FlushInterval.Duration == 0 {
		mc.FlushInterval.Duration = defaultFlushInteval
	}

	go startFlush(mc.FlushInterval.Duration)

	return nil
}

func startFlush(interval time.Duration) {
	for _ = range time.Tick(interval) {
		allRegs := stats.GetAllRegistries()
		// flush each reg to all sinks
		for _, reg := range allRegs {
			for _, sink := range sinks {
				sink.Flush(reg)
			}
		}
	}
}
