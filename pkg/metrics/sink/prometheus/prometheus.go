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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/metrics/sink"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

var (
	sinkType        = "prometheus"
	defaultEndpoint = "/metrics"
	numBufPool      = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, 24)
			return &b
		},
	}
)

func init() {
	sink.RegisterSink(sinkType, builder)
}

// promConfig contains config for all PromSink
type promConfig struct {
	ExportUrl             string    `json:"export_url"` // when this value is not nil, PromSink will work under the PUSHGATEWAY mode.
	Port                  int       `json:"port"`       // pull mode attrs
	Endpoint              string    `json:"endpoint"`
	DisableCollectProcess bool      `json:"disable_collect_process"`
	DisableCollectGo      bool      `json:"disable_collect_go"`
	Percentiles           []int     `json:"percentiles,omitempty"`
	percentilesFloat      []float64 // not config, trans with Percentiles
}

// promSink extract metrics from stats registry with specified interval
type promSink struct {
	config *promConfig

	registry prometheus.Registerer //Prometheus registry
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
func (psink *promSink) Flush(writer io.Writer, ms []types.Metrics) {
	w := writer

	// mark whose TYPE/HELP text already printed
	tracker := make(map[string]bool)
	buf := buffer.GetIoBuffer(256)

	for _, m := range ms {
		typ := m.Type()
		labelKeys, labelVals := m.SortedLabels()
		if sink.IsExclusionLabels(labelKeys) {
			continue
		}

		// TODO cached in metrics struct, avoid calc for each flush
		prefix := typ + "_"
		suffix := makeLabelStr(labelKeys, labelVals)

		m.Each(func(name string, i interface{}) {
			if sink.IsExclusionKeys(name) {
				return
			}
			switch metric := i.(type) {
			case gometrics.Counter:
				psink.flushCounter(tracker, buf, flattenKey(prefix+name), suffix, float64(metric.Count()))
			case gometrics.Gauge:
				psink.flushGauge(tracker, buf, flattenKey(prefix+name), suffix, float64(metric.Value()))
			case gometrics.Histogram:
				psink.flushHistogram(tracker, buf, flattenKey(prefix+name), suffix, metric.Snapshot())
			}
			buf.WriteTo(w)
			buf.Reset()
		})
	}
}

func (psink *promSink) flushHistogram(tracker map[string]bool, buf types.IoBuffer, name string, labels string, snapshot gometrics.Histogram) {
	// min
	psink.flushGauge(tracker, buf, name+"_min", labels, float64(snapshot.Min()))
	// max
	psink.flushGauge(tracker, buf, name+"_max", labels, float64(snapshot.Max()))
	// flush P90 P95 P99 percentiles
	if len(psink.config.Percentiles) == 0 {
		return
	}
	ps := snapshot.Percentiles(psink.config.percentilesFloat)
	if len(ps) == len(psink.config.Percentiles) {
		for i, p := range psink.config.Percentiles {
			psink.flushGauge(tracker, buf, name, fmt.Sprintf("%s,percentile=\"P%d\"", labels, p), ps[i])
		}
	}
}

func (psink *promSink) flushGauge(tracker map[string]bool, buf types.IoBuffer, name string, labels string, val float64) {
	// type
	if !tracker[name] {
		buf.WriteString("# TYPE ")
		buf.WriteString(name)
		buf.WriteString(" gauge\n")
		tracker[name] = true
	}
	// metric
	buf.WriteString(name)
	buf.WriteString("{")
	buf.WriteString(labels)
	buf.WriteString("} ")
	writeFloat(buf, val)
	buf.WriteString("\n")
}

func (psink *promSink) flushCounter(tracker map[string]bool, buf types.IoBuffer, name string, labels string, val float64) {
	// type
	if !tracker[name] {
		buf.WriteString("# TYPE ")
		buf.WriteString(name)
		buf.WriteString(" counter\n")
		tracker[name] = true
	}

	// metric
	buf.WriteString(name)
	buf.WriteString("{")
	buf.WriteString(labels)
	buf.WriteString("} ")
	writeFloat(buf, val)
	buf.WriteString("\n")
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
		real: promhttp.HandlerFor(promReg, promhttp.HandlerOpts{
			DisableCompression: true,
		}),
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", config.Port),
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

	if len(promCfg.Percentiles) > 0 {
		percentilesFloat := make([]float64, 0, len(promCfg.Percentiles))
		for _, p := range promCfg.Percentiles {
			// nolint
			if p > 100 {
				return nil, fmt.Errorf("percentile {%d} must le 100", p)
			}
			percentilesFloat = append(percentilesFloat, float64(p)/100)
		}
		promCfg.percentilesFloat = percentilesFloat
	}

	return NewPromeSink(promCfg), nil
}

// input: keys=[cluster,host] values=[app1,server2]
// output: cluster="app1",host="server"
func makeLabelStr(keys, values []string) (out string) {
	if length := len(keys); length > 0 {
		out = keys[0] + "=\"" + values[0] + "\""
		for i := 1; i < length; i++ {
			out += "," + keys[i] + "=\"" + values[i] + "\""
		}
	}
	return
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

func writeFloat(w types.IoBuffer, f float64) (int, error) {
	switch {
	case f == 1:
		return w.WriteString("1.0")
	case f == 0:
		return w.WriteString("0.0")
	case f == -1:
		return w.WriteString("-1.0")
	case math.IsNaN(f):
		return w.WriteString("NaN")
	case math.IsInf(f, +1):
		return w.WriteString("+Inf")
	case math.IsInf(f, -1):
		return w.WriteString("-Inf")
	default:
		bp := numBufPool.Get().(*[]byte)
		*bp = strconv.AppendFloat((*bp)[:0], f, 'g', -1, 64)
		// Add a .0 if used fixed point and there is no decimal
		// point already. This is for future proofing with OpenMetrics,
		// where floats always contain either an exponent or decimal.
		if !bytes.ContainsAny(*bp, "e.") {
			*bp = append(*bp, '.', '0')
		}
		written, err := w.Write(*bp)
		numBufPool.Put(bp)
		return written, err
	}
}

// Only [a-zA-Z0-9:_] are valid in metric names, any other characters should be sanitized to an underscore.
var flattenRegexp = regexp.MustCompile("[^a-zA-Z0-9_:]")

func flattenKey(key string) string {
	return flattenRegexp.ReplaceAllString(key, "_")
}
