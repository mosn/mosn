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

package metrics

import (
	"testing"

	gometrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"

	"mosn.io/api"
)

func TestNilMetricsIsMetrics(t *testing.T) {
	m, err := NewNilMetrics(MosnMetaType, nil)
	assert.NoError(t, err)
	_, ok := m.(api.Metrics)
	assert.True(t, ok)
}

func TestNilMetricsSuppliers(t *testing.T) {
	m, _ := NewNilMetrics(MosnMetaType, nil)
	c := m.Counter("counter")
	assert.IsType(t, gometrics.NilCounter{}, c)

	g := m.Gauge("gauge")
	assert.IsType(t, gometrics.NilGauge{}, g)

	h := m.Histogram("histogram")
	assert.IsType(t, gometrics.NilHistogram{}, h)

	e := m.EWMA("ewma", 0.1)
	assert.IsType(t, gometrics.NilEWMA{}, e)
}
