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
	"io"
	"github.com/prometheus/common/expfmt"

	dto "github.com/prometheus/client_model/go"
	"github.com/gogo/protobuf/proto"
	"sync"
	"compress/gzip"
)

var (
	sinkType        = "prometheus"
	defaultEndpoint = "/metrics"
	gzipPool        = sync.Pool{
		New: func() interface{} {
			return gzip.NewWriter(nil)
		},
	}
)

func init() {
	sink.RegisterSink(sinkType, builder)
}

// promConfig contains config for all PromSink
type promConfig struct {
	ExportUrl string `json:"export_url"` // when this value is not nil, PromSink will work under the PUSHGATEWAY mode.

	Port     int    `json:"port"` // pull mode attrs
	Endpoint string `json:"endpoint"`

	DisableCollectProcess bool `json:"disable_collect_process"`
	DisableCollectGo      bool `json:"disable_collect_go"`
	DisablePassiveFlush   bool `json:"disable_passive_flush"`
}

// promSink extract metrics from stats registry with specified interval
type promSink struct {
	config *promConfig

	registry   prometheus.Registerer //Prometheus registry
}

type promHttpExporter struct {
	sink *promSink
	real http.Handler
}

func (exporter *promHttpExporter) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	// 1. export process and go metrics
	exporter.real.ServeHTTP(rsp, req)

	// 2. mosn metrics
	exporter.sink.Flush(rsp, metrics.GetAll())
}

// ~ MetricsSink
func (sink *promSink) Flush(writer io.Writer, ms []types.Metrics) {
	format := expfmt.FmtText
	w := writer

	rsp, ok := writer.(http.ResponseWriter)
	if ok {
		format = expfmt.Format(rsp.Header().Get("Content-Type"))

		// gzip
		if rsp.Header().Get("Content-Encoding") == "gzip" {
			gz := gzipPool.Get().(*gzip.Writer)
			defer gzipPool.Put(gz)

			gz.Reset(w)
			defer gz.Close()

			w = gz
		}
	}

	enc := expfmt.NewEncoder(w, format)

	for _, m := range ms {
		typ := m.Type()
		labelKeys, labelVals := m.SortedLabels()

		// TODO cached in metrics struct, avoid calc for each flush
		prefix := strings.Join(labelKeys, "_") + "_" + typ + "_"
		//labels := formatKV(labelKeys, labelVals)
		labels := makeLabelPair(labelKeys, labelVals)

		m.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case gometrics.Counter:
				sink.flushCounter(enc, prefix+name, labels, float64(metric.Count()))
			case gometrics.Gauge:
				sink.flushCounter(enc, prefix+name, labels, float64(metric.Value()))
			case gometrics.Histogram:
				sink.flushHistogram(enc, prefix+name, labels, metric.Snapshot())
			}
		})
	}
}

func (sink *promSink) flushHistogram(enc expfmt.Encoder, name string, labels []*dto.LabelPair, snapshot gometrics.Histogram) {
	// min
	sink.flushGauge(enc, name+"_min", labels, float64(snapshot.Min()))
	// max
	sink.flushGauge(enc, name+"_max", labels, float64(snapshot.Max()))
}

func (sink *promSink) flushGauge(enc expfmt.Encoder, name string, labels []*dto.LabelPair, val float64) {
	enc.Encode(&dto.MetricFamily{
		Name:   proto.String(name),
		Type:   dto.MetricType_GAUGE.Enum(),
		Metric: []*dto.Metric{{Label: labels, Gauge: &dto.Gauge{Value: proto.Float64(val)}}},
	})
}

func (sink *promSink) flushCounter(enc expfmt.Encoder, name string, labels []*dto.LabelPair, val float64) {
	enc.Encode(&dto.MetricFamily{
		Name:   proto.String(name),
		Type:   dto.MetricType_COUNTER.Enum(),
		Metric: []*dto.Metric{{Label: labels, Counter: &dto.Counter{Value: proto.Float64(val)}}},
	})
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
		config:   config,
		registry: promReg,
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

func makeLabelPair(keys, values []string) (pairs []*dto.LabelPair) {
	if length := len(keys); length == len(values) {
		pairs = make([]*dto.LabelPair, length)
		for i := 0; i < length; i++ {
			pairs[i] = &dto.LabelPair{
				Name:  proto.String(keys[i]),
				Value: proto.String(values[i]),
			}
		}
	}
	return
}
