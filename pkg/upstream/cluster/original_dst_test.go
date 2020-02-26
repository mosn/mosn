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
	"mosn.io/api"
	"testing"

	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
	"net"
)

// LbCtx is a types.LoadBalancerContext implementation
type LbCtx struct {
	ctx     context.Context
	cluster types.ClusterInfo
}

func (c *LbCtx) MetadataMatchCriteria() api.MetadataMatchCriteria {
	return nil
}

func (c *LbCtx) DownstreamConnection() net.Conn {
	return nil
}

func (c *LbCtx) DownstreamHeaders() api.HeaderMap {
	return nil
}

func (c *LbCtx) DownstreamContext() context.Context {
	return c.ctx
}

func (c *LbCtx) DownstreamCluster() types.ClusterInfo {
	return c.cluster
}

func TestChooseHost(t *testing.T) {
	hostSet := &hostSet{}
	orilb := newOriginalDstLoadBalancer(hostSet)
	orihost := "127.0.0.1:8888"
	oriRemoteAddr, _ := net.ResolveTCPAddr("", orihost)
	ctx := mosnctx.WithValue(context.Background(), types.ContextOriRemoteAddr, oriRemoteAddr)
	cluster := &clusterInfo{
		name:   "testOriDst",
		lbType: types.ORIGINAL_DST,
	}

	lbCtx := &LbCtx{
		ctx:     ctx,
		cluster: cluster,
	}

	host := orilb.ChooseHost(lbCtx)
	if host.AddressString() != orihost {
		t.Fatalf("expected choose failed, expect host: %s, but got: %s", orihost, host.AddressString())
	}
}
