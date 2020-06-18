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
	"fmt"
	"time"

	"mosn.io/mosn/pkg/cel"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/metrics"
)

var compiler = cel.NewExpressionBuilder(attributemanifest, cel.CompatCEXL)

type metric struct {
	metricType MetricType
	value      attribute.Expression
	dimensions map[string]attribute.Expression
	name       string
}

type Stat struct {
	MetricType MetricType
	Name       string
	Labels     map[string]string
	Value      int64
}

func newMetrics(confs []MetricConfig, definitions []MetricDefinition) ([]*metric, error) {
	ms := make([]*metric, 0, len(confs)*len(definitions))
	for _, definition := range definitions {
		for _, metric := range confs {
			if metric.Name != "" && metric.Name != definition.Name {
				continue
			}
			if len(metric.Dimensions) > metrics.MaxLabelCount {
				return nil, metrics.ErrLabelCountExceeded
			}
			metric, err := newMetric(&metric, &definition)
			if err != nil {
				return nil, err
			}
			ms = append(ms, metric)
		}
	}
	return ms, nil
}

func newMetric(conf *MetricConfig, definition *MetricDefinition) (*metric, error) {
	m := &metric{
		metricType: definition.Type,
		name:       definition.Name,
		dimensions: map[string]attribute.Expression{},
	}
	expr, _, err := compiler.Compile(definition.Value)
	if err != nil {
		return nil, err
	}
	m.value = expr

	for key, temp := range conf.Dimensions {
		expr, _, err := compiler.Compile(temp)
		if err != nil {
			return nil, err
		}
		m.dimensions[key] = expr
	}
	return m, nil
}

func (m *metric) Stat(bag attribute.Bag) (*Stat, error) {
	value, err := m.value.Evaluate(bag)
	if err != nil {
		return nil, err
	}
	var val int64
	switch t := value.(type) {
	case int64:
		val = t
	case time.Duration:
		val = int64(t / time.Millisecond)
	default:
		return nil, fmt.Errorf("is not int64 or time.Duration")
	}

	labels := map[string]string{}
	for key, expr := range m.dimensions {
		v, err := expr.Evaluate(bag)
		if err != nil {
			return nil, err
		}
		var str string
		if s, ok := v.(string); ok {
			str = s
		} else {
			str = fmt.Sprint(v)
		}
		labels[key] = str
	}

	return &Stat{
		MetricType: m.metricType,
		Name:       m.name,
		Labels:     labels,
		Value:      val,
	}, nil
}
