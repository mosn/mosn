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
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

// LbCtx is a types.LoadBalancerContext implementation
type LbCtx struct {
	ctx     context.Context
	cluster types.ClusterInfo
	headers api.HeaderMap
}

func (c *LbCtx) MetadataMatchCriteria() api.MetadataMatchCriteria {
	return nil
}

func (c *LbCtx) DownstreamConnection() net.Conn {
	return nil
}

func (c *LbCtx) DownstreamHeaders() api.HeaderMap {
	return c.headers
}

func (c *LbCtx) DownstreamContext() context.Context {
	return c.ctx
}

func (c *LbCtx) DownstreamCluster() types.ClusterInfo {
	return c.cluster
}

func (c *LbCtx) DownstreamRoute() api.Route {
	return nil
}

type Header struct {
	v map[string]string
}

func (h *Header) Get(key string) (string, bool) {
	k, ok := h.v[key]
	return k, ok
}

func (h *Header) Set(key, value string) {
	h.v[key] = value
}

func (h *Header) Add(key, value string) {
}

func (h *Header) Del(key string) {

}

func (h *Header) Range(f func(key, value string) bool) {

}

func (h *Header) Clone() api.HeaderMap {
	return h
}

func (h *Header) ByteSize() uint64 {
	return 0
}

func TestChooseHost(t *testing.T) {
	// check use original dst
	hostSet := &hostSet{}
	orilb := newOriginalDstLoadBalancer(nil, hostSet)
	orihost := "127.0.0.1:8888"
	oriRemoteAddr, _ := net.ResolveTCPAddr("", orihost)
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableOriRemoteAddr, oriRemoteAddr)
	oriDstCfg := &v2.LBOriDstConfig{
		UseHeader: false,
	}

	cluster := &clusterInfo{
		name:         "testOriDst",
		lbType:       types.ORIGINAL_DST,
		lbOriDstInfo: NewLBOriDstInfo(oriDstCfg),
	}

	lbCtx := &LbCtx{
		ctx:     ctx,
		cluster: cluster,
	}

	host := orilb.ChooseHost(lbCtx)
	if host.AddressString() != orihost {
		t.Fatalf("expected choose failed, expect host: %s, but got: %s", orihost, host.AddressString())
	}

	// check use host header
	oriDstCfg = &v2.LBOriDstConfig{
		UseHeader: true,
	}

	cluster = &clusterInfo{
		name:         "testOriDst",
		lbType:       types.ORIGINAL_DST,
		lbOriDstInfo: NewLBOriDstInfo(oriDstCfg),
	}

	lbCtx = &LbCtx{
		ctx:     ctx,
		cluster: cluster,
		headers: &Header{
			v: make(map[string]string),
		},
	}

	orihost = "127.0.0.1:9999"
	lbCtx.headers.Set("host", orihost)

	host = orilb.ChooseHost(lbCtx)
	if host.AddressString() != orihost {
		t.Fatalf("expected choose failed, expect host: %s, but got: %s", orihost, host.AddressString())
	}

	// check default port 80, use header
	lbCtx = &LbCtx{
		ctx:     ctx,
		cluster: cluster,
		headers: &Header{
			v: map[string]string{
				"host": "127.0.0.1", // without port
			},
		},
	}
	host = orilb.ChooseHost(lbCtx)
	require.Equal(t, "127.0.0.1:80", host.AddressString())

	// check replace by local
	oriDstCfg = &v2.LBOriDstConfig{
		UseHeader:    true,
		ReplaceLocal: true,
	}
	cluster = &clusterInfo{
		name:         "testOriDs_replace",
		lbType:       types.ORIGINAL_DST,
		lbOriDstInfo: NewLBOriDstInfo(oriDstCfg),
	}
	lbCtx = &LbCtx{
		ctx:     ctx,
		cluster: cluster,
		headers: &Header{
			v: map[string]string{
				"host": "192.168.1.1:9080",
			},
		},
	}
	host = orilb.ChooseHost(lbCtx)
	require.Equal(t, "127.0.0.1:9080", host.AddressString())
}
