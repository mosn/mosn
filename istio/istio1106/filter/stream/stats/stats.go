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

package stats

import (
	"context"
	"encoding/json"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/istio/istio1106/config/v2"
	"mosn.io/mosn/pkg/cel/extract"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/pkg/buffer"
)

func init() {
	api.RegisterStream(v2.IstioStats, CreateStatsFilterFactory)
}

// FilterConfigFactory filter config factory
type FilterConfigFactory struct {
	prefix  string
	metrics *Metrics
}

type statsFilter struct {
	context               context.Context
	receiverFilterHandler api.StreamReceiverFilterHandler

	prefix  string
	metrics *Metrics

	buf      buffer.IoBuffer
	trailers api.HeaderMap
}

// newStatsFilter used to create new stats filter
func newStatsFilter(ctx context.Context, prefix string, metrics *Metrics) *statsFilter {
	filter := &statsFilter{
		context: ctx,
		prefix:  prefix,
		metrics: metrics,
	}
	return filter
}

func (f *statsFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	f.buf = buf
	f.trailers = trailers
	return api.StreamFilterContinue
}

func (f *statsFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.receiverFilterHandler = handler
}

func (f *statsFilter) OnDestroy() {}

func (f *statsFilter) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	if reqHeaders == nil || respHeaders == nil || requestInfo == nil || f.metrics == nil || len(f.metrics.definitions) == 0 {
		return
	}

	attributes := extract.ExtractAttributes(ctx, reqHeaders, respHeaders, requestInfo, f.buf, f.trailers, time.Now())
	stats, err := f.metrics.Stat(attributes)
	if err != nil {
		log.DefaultLogger.Errorf("stats error: %s", err.Error())
		return
	}
	for _, stat := range stats {
		err = updateMetric(stat.MetricType, f.prefix, stat.Name, stat.Labels, stat.Value)
		if err != nil {
			log.DefaultLogger.Errorf("stats update error: %s", err.Error())
			continue
		}
	}
}

// CreateFilterChain for create stats filter
func (f *FilterConfigFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	filter := newStatsFilter(context, f.prefix, f.metrics)
	callbacks.AddStreamReceiverFilter(filter, api.AfterRoute)
	callbacks.AddStreamAccessLog(filter)
}

// CreateStatsFilterFactory for create stats filter factory
func CreateStatsFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	data, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}
	c := StatsConfig{}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}

	if len(c.Metrics) == 0 {
		c.Metrics = defaultMetricConfig
	}

	if len(c.Definitions) == 0 {
		c.Definitions = defaultMetricDefinition
	}

	ms, err := newMetrics(c.Metrics, c.Definitions)
	if err != nil {
		return nil, err
	}

	return &FilterConfigFactory{metrics: ms, prefix: c.StatPrefix}, nil
}

func updateMetric(mt MetricType, prefix, name string, labels map[string]string, val int64) error {
	met, err := metrics.NewMetrics(prefix, labels)
	if err != nil {
		return err
	}
	switch mt {
	case MetricTypeCounter, "":
		met.Counter(name).Inc(val)
	case MetricTypeGauge:
		met.Gauge(name).Update(val)
	case MetricTypeHistogram:
		met.Histogram(name).Update(val)
	}
	return nil
}
