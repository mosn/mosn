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
	"context"
	"sync"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	str "github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
)

func init() {
	log.InitDefaultLogger("", log.DEBUG)
}

type mockClient struct {
	t *testing.T
}

func newMockClient(t *testing.T) *mockClient {
	c := &mockClient{
		t: t,
	}
	return c
}

func (c *mockClient) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
	log.DefaultLogger.Debugf("data:%v endStream:%v", data, endStream)
}
func (c *mockClient) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
	log.DefaultLogger.Debugf("trailers:%v", trailers)
}
func (c *mockClient) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
	c.t.Errorf("err:%v headers:%v", err, headers)
}
func (c *mockClient) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	log.DefaultLogger.Debugf("headers:%v endStream:%v", headers, endStream)
}

func checkNumbers(t *testing.T, codecClient str.CodecClient, want int) {
	num := codecClient.ActiveRequestsNum()
	if num != want {
		t.Fatalf("activeRequestsNum:%d want:%d", num, want)
	}
}

func TestActiveRequests(t *testing.T) {
	cli := newMockClient(t)
	host := cluster.NewHost(v2.Host{
		HostConfig: v2.HostConfig{
			Address: "127.0.0.1", Hostname: "test", Weight: 0,
		},
	}, cluster.NewClusterInfo())
	ac := &activeClient{
		pool: &connPool{host: host},
	}
	codecClient := NewHTTP1CodecClient(context.Background(), ac)
	ctx := context.Background()

	codecClient.NewStream(ctx, protocol.StreamIDConv(1), cli)

	checkNumbers(t, codecClient, 1)
	codecClient.NewStream(ctx, protocol.StreamIDConv(2), cli)
	checkNumbers(t, codecClient, 2)

	codecClient.OnEvent(types.Connected)
	checkNumbers(t, codecClient, 2)
	// clear all
	codecClient.OnEvent(types.RemoteClose)
	checkNumbers(t, codecClient, 0)
	// data race
	wg := sync.WaitGroup{}
	cnt := 1000
	gps := 10
	wg.Add(gps)
	for i := 0; i < gps; i++ {
		go func(pre int) {
			defer wg.Done()
			for j := 0; j < cnt; j++ {
				codecClient.NewStream(ctx, protocol.StreamIDConv(uint32(pre*cnt+j)), cli)
				codecClient.ActiveRequestsNum()
				codecClient.OnEvent(types.LocalClose)
			}
		}(i)
	}
	wg.Wait()
}
