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
)

var compiler = cel.NewExpressionBuilder(attributemanifest, cel.CompatCEXL)

type metric struct {
	value      attribute.Expression
	dimensions map[string]attribute.Expression
	name       string
}

type Stat struct {
	Name   string
	Labels map[string]string
	Value  int64
}

func newMetric(conf *MetricConfig) (*metric, error) {
	m := &metric{
		name:       conf.Name,
		dimensions: map[string]attribute.Expression{},
	}
	expr, _, err := compiler.Compile(conf.Value)
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
		Name:   m.name,
		Labels: labels,
		Value:  val,
	}, nil
}
