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

	"github.com/stretchr/testify/assert"
)

func TestDownstreamMetrics(t *testing.T) {
	proxyMetrics := NewProxyStats("test_proxy")

	counter := proxyMetrics.Counter("request_count")
	counter.Inc(100)
	assert.Equal(t, counter.Count(), int64(100))
}

func TestHealthStats(t *testing.T) {
	healthStats := NewHealthStats("test_service")
	counter := healthStats.Counter("upup")
	counter.Inc(999)
	assert.Equal(t, counter.Count(), int64(999))
}

func TestMosnMetrics(t *testing.T) {
	FlushMosnMetrics = true
	defer func() {
		FlushMosnMetrics = false
	}()

	m := NewMosnMetrics()
	c := m.Counter("mosn_mosn")
	c.Inc(100)
	c.Dec(10)
	assert.Equal(t, c.Count(), int64(90))
}

func TestUseless(t *testing.T) {
	// useless functions test
	SetGoVersion("go1.19")
	SetVersion("mosn1.13")
	SetStateCode(1)
	AddListenerAddr("localhost:1111")
}
