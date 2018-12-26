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

package http

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
	metrics "github.com/rcrowley/go-metrics"
)

type HTTPStatusStats struct {
	Host    *httpStatusStats
	Cluster *httpStatusStats
}

type httpStatusStats struct {
	HTTPStatus2XX metrics.Counter
	HTTPStatus3XX metrics.Counter
	HTTPStatus4XX metrics.Counter
	HTTPStatus5XX metrics.Counter
}

// NewHTTPStatusStats makes a HTTPStatusStats
// the stats should have same label with host/cluster
func NewHTTPStatusStats(host types.Host) *HTTPStatusStats {
	clusterName := host.ClusterInfo().Name()
	hostAddr := host.AddressString()
	clusterStats := stats.NewClusterStats(clusterName)
	hostStats := stats.NewHostStats(clusterName, hostAddr)
	return &HTTPStatusStats{
		Host: &httpStatusStats{
			HTTPStatus2XX: hostStats.Counter(stats.HTTPStatus2XX),
			HTTPStatus3XX: hostStats.Counter(stats.HTTPStatus3XX),
			HTTPStatus4XX: hostStats.Counter(stats.HTTPStatus4XX),
			HTTPStatus5XX: hostStats.Counter(stats.HTTPStatus5XX),
		},
		Cluster: &httpStatusStats{
			HTTPStatus2XX: clusterStats.Counter(stats.HTTPStatus2XX),
			HTTPStatus3XX: clusterStats.Counter(stats.HTTPStatus3XX),
			HTTPStatus4XX: clusterStats.Counter(stats.HTTPStatus4XX),
			HTTPStatus5XX: clusterStats.Counter(stats.HTTPStatus5XX),
		},
	}
}
