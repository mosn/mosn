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

package conv

import (
	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/metrics"
)

// DownstreamType represents downstream  metrics type
const XdsType = "xds"

// metrics key in listener/proxy
const (
	CdsUpdateSuccessTotal = "cds_update_success"
	CdsUpdateRejectTotal  = "cds_update_reject"
	LdsUpdateSuccessTotal = "ls_update_success"
	LdsUpdateRejectTotal  = "lds_update_reject"
)

type XdsStats struct {
	CdsUpdateSuccess gometrics.Counter
	CdsUpdateReject  gometrics.Counter
	LdsUpdateSuccess gometrics.Counter
	LdsUpdateReject  gometrics.Counter
}

// NewXdsStats returns a stats with namespace prefix proxy
func NewXdsStats() XdsStats {
	m, _ := metrics.NewMetrics(XdsType, map[string]string{
		"xds": "info",
	})
	stats := XdsStats{
		CdsUpdateSuccess: m.Counter(CdsUpdateSuccessTotal),
		CdsUpdateReject:  m.Counter(CdsUpdateRejectTotal),
		LdsUpdateSuccess: m.Counter(LdsUpdateSuccessTotal),
		LdsUpdateReject:  m.Counter(LdsUpdateRejectTotal),
	}
	return stats
}
