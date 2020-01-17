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

// FlushMosnMetrics marks output mosn information metrics or not, default is false
var FlushMosnMetrics bool

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
