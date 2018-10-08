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

package healthcheck

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
)

// StartSofaHeartBeat use for hearth-beat starting for sofa bolt in the same codecClient
// for bolt heartbeat, timeout: 90s interval: 15s
func StartSofaHeartBeat(timeout time.Duration, interval time.Duration, hostAddr string,
	codecClient stream.CodecClient, nameHB string, pro sofarpc.ProtocolType) types.HealthCheckSession {

	hcV2 := v2.HealthCheck{
		Timeout:  timeout,
		Interval: interval,
	}
	hcV2.ServiceName = nameHB

	hostV2 := v2.Host{}
	hostV2.Address = hostAddr

	host := cluster.NewHost(hostV2, nil)
	baseHc := newHealthChecker(hcV2)

	hc := newSofaRPCHealthCheckerWithBaseHealthChecker(baseHc, pro)
	hcs := hc.newSofaRPCHealthCheckSession(codecClient, host)
	hcs.Start()

	return hcs
}
