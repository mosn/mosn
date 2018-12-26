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

package sofarpc

import (
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
	metrics "github.com/rcrowley/go-metrics"
)

type SofaRPCStatusStats struct {
	Host    *sofaRPCStatusStats
	Cluster *sofaRPCStatusStats
}

type sofaRPCStatusStats struct {
	SofaRPCSuccess metrics.Counter
	SofaRPCFailed  metrics.Counter
}

// NewSofaRPCStatusStats makes a SofaRPCStatusStats
// the stats should have same label with host/cluster
func NewSofaRPCStatusStats(host types.Host) *SofaRPCStatusStats {
	clusterName := host.ClusterInfo().Name()
	hostAddr := host.AddressString()
	clusterStats := stats.NewClusterStats(clusterName)
	hostStats := stats.NewHostStats(clusterName, hostAddr)
	return &SofaRPCStatusStats{
		Host: &sofaRPCStatusStats{
			SofaRPCSuccess: hostStats.Counter(stats.SofaRPCSuccess),
			SofaRPCFailed:  hostStats.Counter(stats.SofaRPCFailed),
		},
		Cluster: &sofaRPCStatusStats{
			SofaRPCSuccess: clusterStats.Counter(stats.SofaRPCSuccess),
			SofaRPCFailed:  clusterStats.Counter(stats.SofaRPCFailed),
		},
	}
}
