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
	"context"
	"sync"

	"mosn.io/mosn/pkg/types"
)

// clusterConnectionPool is a connection pool wrapper that contains
// an extra flag to make whether the connection pool is support tls.
// the connection pool hash value function is overwrited.
type clusterConnectionPool struct {
	types.ConnectionPool
	rwmutex    sync.RWMutex
	hash       *types.HashValue
	supportTLS bool
}

func createConnectionPool(balancerContext types.LoadBalancerContext, host types.Host, factory func(ctx context.Context, host types.Host) types.ConnectionPool) *clusterConnectionPool {
	pool := factory(balancerContext.DownstreamContext(), host)
	return &clusterConnectionPool{
		ConnectionPool: pool,
		hash:           pool.TLSHashValue(),
		supportTLS:     host.SupportTLS(),
	}
}

func (connpool *clusterConnectionPool) TLSHashValue() *types.HashValue {
	connpool.rwmutex.RLock()
	defer connpool.rwmutex.RUnlock()
	return connpool.hash
}

func (connpool *clusterConnectionPool) UpdateTLSHashValue(hash *types.HashValue) {
	connpool.rwmutex.Lock()
	defer connpool.rwmutex.Unlock()
	connpool.hash = hash
}

func (connpool *clusterConnectionPool) SupportTLS() bool {
	return connpool.supportTLS
}
