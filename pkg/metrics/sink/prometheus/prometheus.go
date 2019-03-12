/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package prometheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/metrics/sink"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/alipay/sofa-mosn/pkg/metrics"
)

func init() {
	sink.RegisterSink("prometheus", builder)
}

var (
	defaultEndpoint = "/metrics"
)

// promConfig contains config for all PromSink
type promConfig struct {
	ExportUrl string `json:"export_url"` // when this value is not nil, PromSink will work under the PUSHGATEWAY mode.

	Port     int    `json:"port"` // pull mode attrs
	Endpoint string `json:"endpoint"`

	DisableCollectProcess bool `json:"disable_collect_process"`
	DisableCollectGo      bool `json:"disable_collect_go"`
}

// promSink extract metrics from stats registry with specified interval
type promSink struct {
	config *promConfig

	registry  prometheus.Registerer //Prometheus registry
	gaugeVecs map[string]*prometheus.GaugeVec
}

type promHttpExporter struct {
	sink *promSink
	real http.Handler
}

func (exporter *promHttpExporter) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	// 1. flush metrics
	exporter.sink.Flush(metrics.GetAll())

	// 2. export
	exporter.real.ServeHTTP(rsp, req)
}

// ~ MetricsSink
func (sink *promSink) Flush(ms []types.Metrics) {
	for _, m := range ms {
		typ := m.Type()
		labelKeys, labelVals := m.SortedLabels()
		m.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case gometrics.Counter:
				sink.gauge(typ, labelKeys, labelVals, name, float64(metric.Count()))
			case gometrics.Gauge:
				sink.gauge(typ, labelKeys, labelVals, name, float64(metric.Value()))
			case gometrics.Histogram:
				snap := metric.Snapshot()
				sink.histogramVec(typ, labelKeys, labelVals, name, snap)
			}
		})
	}
}

// NewPromeSink returns a metrics sink that produces Prometheus metrics using store data
func NewPromeSink(config *promConfig) types.MetricsSink {
	promReg := prometheus.NewRegistry()
	// register process and  go metrics
	if !config.DisableCollectProcess {
		promReg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}
	if !config.DisableCollectGo {
		promReg.MustRegister(prometheus.NewGoCollector())
	}

	promSink := &promSink{
		config:    config,
		registry:  promReg,
		gaugeVecs: make(map[string]*prometheus.GaugeVec),
	}

	// export http for prometheus
	srvMux := http.NewServeMux()
	srvMux.Handle(config.Endpoint, &promHttpExporter{
		sink: promSink,
		real: promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}),
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: srvMux,
	}

	store.AddService(srv, "prometheus", nil, nil)

	return promSink
}

func (sink *promSink) histogramVec(typ string, labelKeys, labelVals []string, name string, snap gometrics.Histogram) {
	sink.histogramVecWithValue(typ, labelKeys, labelVals, name+"_max", float64(snap.Max()))
	sink.histogramVecWithValue(typ, labelKeys, labelVals, name+"_min", float64(snap.Min()))
}

func (sink *promSink) histogramVecWithValue(typ string, labelKeys, labelVals []string, name string, value float64) {
	namespace := strings.Join(labelKeys, "_")
	key := namespace + "_" + typ + "_" + name
	g, ok := sink.gaugeVecs[key]
	if !ok {
		g = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: flattenKey(namespace),
			Subsystem: flattenKey(typ),
			Name:      flattenKey(name),
			Help:      "histogram metrics",
		}, labelKeys)
		sink.registry.MustRegister(g)
		sink.gaugeVecs[key] = g
	}
	g.WithLabelValues(labelVals...).Set(value)
}

func (sink *promSink) gauge(typ string, labelKeys, labelVals []string, name string, val float64) {
	namespace := strings.Join(labelKeys, "_")
	key := namespace + "_" + typ + "_" + name
	g, ok := sink.gaugeVecs[key]
	if !ok {
		g = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: flattenKey(namespace),
			Subsystem: flattenKey(typ),
			Name:      name,
		}, labelKeys)

		sink.registry.MustRegister(g)
		sink.gaugeVecs[key] = g
	}
	g.WithLabelValues(labelVals...).Set(val)
}

// factory
func builder(cfg map[string]interface{}) (types.MetricsSink, error) {
	// parse config
	promCfg := &promConfig{}

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
		return nil, errors.New("prometheus sink's port is not specified")
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

func flattenKey(key string) string {
	key = strings.Replace(key, " ", "_", -1)
	key = strings.Replace(key, ".", "_", -1)
	key = strings.Replace(key, "-", "_", -1)
	key = strings.Replace(key, "=", "_", -1)
	return key
}
