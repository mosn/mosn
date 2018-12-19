package sink

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"time"
	"github.com/alipay/sofa-mosn/pkg/stats"
)

var defaultFlushInteval = time.Second

func StartFlush(sinks []types.MetricsSink, interval time.Duration) {
	if interval <= 0 {
		interval = defaultFlushInteval
	}

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
