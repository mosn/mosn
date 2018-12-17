package sink

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
	"github.com/rcrowley/go-metrics"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
)

// PromConfig contains config for all PromSink
type PromConfig struct {
	ExportUrl     string        // when this value is not nil, PromSink will work under the PUSHGATEWAY mode.
	FlushInterval time.Duration //interval to update prom metrics
}

// PromSink extract metrics from stats registry with specified interval
type PromSink struct {
	config *PromConfig

	registry  prometheus.Registerer //Prometheus registry
	gauges    map[string]prometheus.Gauge
	gaugeVecs map[string]prometheus.GaugeVec
}

// ~ MetricsSink
func (sink *PromSink) Flush(registry metrics.Registry) {
	registry.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case metrics.Counter:
			//fmt.Fprintf(os.Stderr, "Counter: %s %f\n", name, float64(metric.Count()))
			sink.gauge(name, float64(metric.Count()))
		case metrics.Gauge:
			//fmt.Fprintf(os.Stderr, "Gauge: %s %d\n", name, metric.Value())
			sink.gauge(name, float64(metric.Value()))
		case metrics.GaugeFloat64:
			//fmt.Fprintf(os.Stderr, "GaugeFloat64: %s %f\n", name, metric.Value())
			sink.gauge(name, float64(metric.Value()))
		case metrics.Histogram:
			snap := metric.Snapshot()
			sink.histogramVec(name, snap)
		case metrics.Meter:
			snap := metric.Snapshot()
			sink.meterVec(name, snap)
		case metrics.Timer:
			lastSample := metric.Snapshot().Rate1()
			sink.gauge(name, float64(lastSample))
		}
	})
}

// NewPrometheusProvider returns a Provider that produces Prometheus metrics.
// Namespace and subsystem are applied to all produced metrics.
func NewPromeSink(config *PromConfig) types.MetricsSink {
	promReg := prometheus.NewRegistry()
	// register process and  go metrics
	//promReg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	//promReg.MustRegister(prometheus.NewGoCollector())

	// export http for prometheus
	go func () {
		http.Handle("/metrics", promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	return &PromSink{
		config:    config,
		registry:  promReg,
		gauges:    make(map[string]prometheus.Gauge),
		gaugeVecs: make(map[string]prometheus.GaugeVec),
	}
}

func (sink *PromSink) meterVec(name string, snap metrics.Meter) {
	g, ok := sink.gaugeVecs[name]
	if !ok {
		statsType, ns, n := stats.KeySplit(name)
		g = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: statsType,
			Subsystem: ns,
			Name:      n,
			Help:      name,
		},
			[]string{
				"type",
			},
		)
		sink.registry.MustRegister(g)
		sink.gaugeVecs[name] = g
	}

	g.WithLabelValues("count").Set(float64(snap.Count()))
	g.WithLabelValues("rate1").Set(snap.Rate1())
	g.WithLabelValues("rate5").Set(snap.Rate5())
	g.WithLabelValues("rate15").Set(snap.Rate15())
	g.WithLabelValues("rate_mean").Set(snap.RateMean())
}

func (sink *PromSink) histogramVec(name string, snap metrics.Histogram) {
	g, ok := sink.gaugeVecs[name]
	if !ok {
		statsType, ns, n := stats.KeySplit(name)
		g = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: statsType,
			Subsystem: ns,
			Name:      n,
			Help:      name,
		},
			[]string{
				"type",
			},
		)
		sink.registry.MustRegister(g)
		sink.gaugeVecs[name] = g
	}
	g.WithLabelValues("count").Set(float64(snap.Count()))
	g.WithLabelValues("max").Set(float64(snap.Max()))
	g.WithLabelValues("min").Set(float64(snap.Min()))
	g.WithLabelValues("mean").Set(snap.Mean())
	g.WithLabelValues("stddev").Set(snap.StdDev())
	g.WithLabelValues("perc75").Set(snap.Percentile(float64(75)))
	g.WithLabelValues("perc95").Set(snap.Percentile(float64(95)))
	g.WithLabelValues("perc99").Set(snap.Percentile(float64(99)))
	g.WithLabelValues("perc999").Set(snap.Percentile(float64(99.9)))
}

func (sink *PromSink) gauge(name string, val float64) {
	g, ok := sink.gauges[name]
	if !ok {
		statsType, ns, n := stats.KeySplit(name)
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: statsType,
			Subsystem: ns,
			Name:      n,
			Help:      name,
		})
		sink.registry.MustRegister(g)
		sink.gauges[name] = g
	}
	g.Set(val)
}
