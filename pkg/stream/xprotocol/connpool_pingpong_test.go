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

package xprotocol

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosn.io/api"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func TestPingPong(t *testing.T) {
	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	ctx := variable.NewVariableContext(context.Background())

	cl := basicCluster(addr, []string{addr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := connpool{
		protocol: api.ProtocolName(dubbo.ProtocolName),
		tlsHash:  &types.HashValue{},
		codec:    &dubbo.XCodec{},
	}
	p.host.Store(host)

	pMultiplex := NewPoolPingPong(&p)
	pInst := pMultiplex.(*poolPingPong)
	var xsList []*xStream
	for i := 0; i < 10; i++ {
		_, sender, failReason := pInst.NewStream(ctx, &receiver{})
		assert.Equal(t, types.PoolFailureReason(""), failReason)
		xs := sender.(*xStream)
		xs.direction = stream.ServerStream
		xsList = append(xsList, xs)
	}

	// destroy all streams
	// these connections should all go back go idleClients
	for i := 0; i < len(xsList); i++ {
		xsList[i].AppendHeaders(context.TODO(), &dubbo.Frame{Header: dubbo.Header{
			Magic:     []byte{1, 2},
			Direction: 0,
		}}, true)
	}

	assert.Equal(t, len(xsList), len(pInst.idleClients))

}

func TestPingPongBoundary(t *testing.T) {
	p := connpool{
		protocol: api.ProtocolName(dubbo.ProtocolName),
		tlsHash:  &types.HashValue{},
		codec:    &dubbo.XCodec{},
	}
	// mock a host
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	host := mock.NewMockHost(ctrl)
	p.host.Store(host)
	// set mock host info
	mockClusterInfo := mock.NewMockClusterInfo(ctrl)
	s1 := metrics.NewClusterStats("test_clustername")
	mockClusterInfo.EXPECT().Stats().Return(&types.ClusterStats{
		UpstreamRequestPendingOverflow: s1.Counter(metrics.UpstreamRequestPendingOverflow),
		UpstreamRequestFailureEject:    s1.Counter(metrics.UpstreamRequestFailureEject),
		UpstreamRequestLocalReset:      s1.Counter(metrics.UpstreamRequestLocalReset),
		UpstreamRequestRemoteReset:     s1.Counter(metrics.UpstreamRequestRemoteReset),
	}).AnyTimes()
	host.EXPECT().ClusterInfo().Return(mockClusterInfo).AnyTimes()
	s := metrics.NewHostStats("test_clustername", "test_host_addr")
	host.EXPECT().HostStats().Return(&types.HostStats{
		UpstreamRequestPendingOverflow: s.Counter(metrics.UpstreamRequestPendingOverflow),
		UpstreamRequestFailureEject:    s.Counter(metrics.UpstreamRequestFailureEject),
		UpstreamRequestLocalReset:      s.Counter(metrics.UpstreamRequestLocalReset),
		UpstreamRequestRemoteReset:     s.Counter(metrics.UpstreamRequestRemoteReset),
	}).AnyTimes()

	// test
	pp := NewPoolPingPong(&p)
	require.True(t, pp.CheckAndInit(context.Background()))
	// set mock call
	t.Run("test overflow", func(t *testing.T) {
		mockResourceMng := mock.NewMockResourceManager(ctrl)
		mockResource := mock.NewMockResource(ctrl)
		mockResourceMng.EXPECT().Requests().Return(mockResource)
		mockResource.EXPECT().CanCreate().Return(false)
		mockClusterInfo.EXPECT().ResourceManager().Return(mockResourceMng)

		_, _, reason := pp.NewStream(context.Background(), &receiver{})
		require.Equal(t, types.Overflow, reason)
	})

	t.Run("test on reset stream", func(t *testing.T) {
		atciveP := &activeClientPingPong{
			pool: pp.(*poolPingPong),
		}
		atciveP.OnResetStream(types.StreamConnectionTermination)
		atciveP.OnResetStream(types.StreamLocalReset)
		atciveP.OnResetStream(types.StreamRemoteReset)
	})

}

type receiver struct{}

func (r *receiver) OnReceive(ctx context.Context, headers api.HeaderMap,
	data buffer.IoBuffer, trailers api.HeaderMap) {
	fmt.Println("receive data")
}

func (r *receiver) OnDecodeError(ctx context.Context, err error,
	headers api.HeaderMap) {
	fmt.Println("decode error")
}
