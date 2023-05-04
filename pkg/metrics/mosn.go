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
	gometrics "github.com/rcrowley/go-metrics"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// MosnMetaType represents mosn basic metrics type
const MosnMetaType = "meta"

// mosn basic info metrics
const (
	GoVersion    = "go_version:"
	Version      = "version:"
	ListenerAddr = "listener_address:"
	StateCode    = "mosn_state_code"
)

var (
	// FlushMosnMetrics marks output mosn information metrics or not, default is false
	FlushMosnMetrics bool

	// LazyFlushMetrics marks flush data with lazy mode
	LazyFlushMetrics bool
)

type SampleType string

const (
	SampleUniform  SampleType = "UNIFORM"
	SampleExpDecay SampleType = "EXP_DECAY"
)

const (
	defaultSampleType    = SampleUniform
	defaultSampleSize    = 100
	defaultExpDecayAlpha = 0.5
)

var (
	sampleType    = defaultSampleType
	sampleSize    = defaultSampleSize
	expDecayAlpha = defaultExpDecayAlpha
)

type SampleFactory func() gometrics.Sample

var sampleFactories = make(map[SampleType]SampleFactory) //nolint:gochecknoglobals

func RegisterSampleFactory(t SampleType, factory SampleFactory) {
	sampleFactories[t] = factory
}

func init() {
	RegisterSampleFactory(SampleUniform, func() gometrics.Sample {
		return gometrics.NewUniformSample(sampleSize)
	})
	RegisterSampleFactory(SampleExpDecay, func() gometrics.Sample {
		return gometrics.NewExpDecaySample(sampleSize, expDecayAlpha)
	})
}

// NewMosnMetrics returns the basic metrics for mosn
// export the function for extension
// multiple calls will only make a metrics object
func NewMosnMetrics() types.Metrics {
	if !FlushMosnMetrics {
		metrics, _ := NewNilMetrics(MosnMetaType, nil)
		return metrics
	}
	metrics, _ := NewMetrics(MosnMetaType, map[string]string{"mosn": "info"})
	return metrics
}

// SetGoVersion set the go version
func SetGoVersion(version string) {
	NewMosnMetrics().Gauge(GoVersion + version).Update(1)
}

// SetVersion set the mosn's version
func SetVersion(version string) {
	NewMosnMetrics().Gauge(Version + version).Update(1)
}

// SetStateCode set the mosn's running state's code
func SetStateCode(code int64) {
	NewMosnMetrics().Gauge(StateCode).Update(code)
}

// AddListenerAddr adds a listener addr info
func AddListenerAddr(addr string) {
	NewMosnMetrics().Gauge(ListenerAddr + addr).Update(1)
}

// SetMetricsFeature enabled metrics feature
func SetMetricsFeature(flushMosn, lazyFlush bool) {
	FlushMosnMetrics = flushMosn
	LazyFlushMetrics = lazyFlush
}

// SetSampleType set sample type for Histogram
func SetSampleType(t SampleType) {
	if _, ok := sampleFactories[t]; !ok {
		if log.DefaultLogger.GetLogLevel() >= log.WARN {
			log.DefaultLogger.Warnf("[metrics] Unknown sample type %s, will use UNIFORM as default", t)
		}

		t = SampleUniform
	}

	sampleType = t
}

// SetSampleSize set sample size for Histogram
func SetSampleSize(size int) {
	sampleSize = size
}

// SetExpDecayAlpha set alpha of ExpDecaySample for Histogram
func SetExpDecayAlpha(alpha float64) {
	expDecayAlpha = alpha
}
