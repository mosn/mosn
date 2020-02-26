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

package cluster

import (
	"sync"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
	"net"
)

func init() {
	RegisterLBType(types.ORIGINAL_DST, newOriginalDstLoadBalancer)
}

// LoadBalancer Implementations
type OriginalDstLoadBalancer struct {
	mutex sync.Mutex
	hosts types.HostSet
	host  map[string]types.Host
}

func newOriginalDstLoadBalancer(hosts types.HostSet) types.LoadBalancer {
	return &OriginalDstLoadBalancer{
		hosts: hosts,
		host:  make(map[string]types.Host),
	}
}

func (lb *OriginalDstLoadBalancer) ChooseHost(lbCtx types.LoadBalancerContext) types.Host {

	// TODO support use_http_header_
	ctx := lbCtx.DownstreamContext()
	cluster := lbCtx.DownstreamCluster()

	oriRemoteAddr := mosnctx.Get(ctx, types.ContextOriRemoteAddr)
	if oriRemoteAddr == nil {
		return nil
	}
	ori := oriRemoteAddr.(*net.TCPAddr)
	ipadd := ori.String()

	var config v2.Host
	config.Address = ipadd
	config.Hostname = ipadd

	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	host, ok := lb.host[ipadd]
	if !ok {
		host = NewSimpleHost(config, cluster)
		lb.host[config.Address] = host
	}

	return host
}

func (lb *OriginalDstLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return true
}

func (lb *OriginalDstLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return 1
}
