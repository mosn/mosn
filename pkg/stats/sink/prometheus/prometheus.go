package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/stats"
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"encoding/json"
	"fmt"
	"github.com/kataras/iris/core/errors"
	"github.com/alipay/sofa-mosn/pkg/stats/sink"
	"strings"
)

func init() {
	sink.RegisterSink("prometheus", builder)
}

var (
	defaultPort     = 8088
	defaultEndpoint = "/metrics"
)

// PromConfig contains config for all PromSink
type PromConfig struct {
	ExportUrl string `json:"export_url"` // when this value is not nil, PromSink will work under the PUSHGATEWAY mode.

	Port     int    `json:"port"` // pull mode attrs
	Endpoint string `json:"endpoint"`

	DisableCollectProcess bool `json:"disable_collect_process"`
	DisableCollectGo      bool `json:"disable_collect_go"`
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
	if !config.DisableCollectProcess {
		promReg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}
	if !config.DisableCollectGo {
		promReg.MustRegister(prometheus.NewGoCollector())
	}

	// export http for prometheus
	go func() {
		http.Handle(config.Endpoint, promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}))
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
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
			Namespace: flattenKey(statsType),
			Subsystem: flattenKey(ns),
			Name:      flattenKey(n),
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
			Namespace: flattenKey(statsType),
			Subsystem: flattenKey(ns),
			Name:      flattenKey(n),
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
			Namespace: flattenKey(statsType),
			Subsystem: flattenKey(ns),
			Name:      flattenKey(n),
			Help:      name,
		})
		sink.registry.MustRegister(g)
		sink.gauges[name] = g
	}
	g.Set(val)
}

func flattenKey(key string) string {
	key = strings.Replace(key, " ", "_", -1)
	key = strings.Replace(key, ".", "_", -1)
	key = strings.Replace(key, "-", "_", -1)
	key = strings.Replace(key, "=", "_", -1)
	return key
}

// factory
func builder(cfg map[string]interface{}) (types.MetricsSink, error) {
	// parse config
	promCfg := &PromConfig{}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing prometheus sink error, err: %v, cfg: %v", err, cfg)
	}
	if err := json.Unmarshal(data, promCfg); err != nil {
		return nil, fmt.Errorf("parsing prometheus sink error, err: %v, cfg: %v", err, cfg)
	}

	if promCfg.ExportUrl != "" {
		return nil, errors.New("prometheus PushGateway mode currently unsupported")
	}

	if promCfg.Port == 0 {
		promCfg.Port = defaultPort
	}

	if promCfg.Endpoint == "" {
		promCfg.Endpoint = defaultEndpoint
	} else {
		if !strings.HasPrefix(promCfg.Endpoint, "/") {
			return nil, fmt.Errorf("invalid endpoint format:%s", promCfg.Endpoint)
		}
	}

	return NewPromeSink(promCfg), nil
}
