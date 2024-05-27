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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	str "mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/variable"
)

type testStats struct {
	success uint32
	timeout uint32
}

func (s *testStats) Record(status types.KeepAliveStatus) {
	switch status {
	case types.KeepAliveSuccess:
		atomic.AddUint32(&s.success, 1)
	case types.KeepAliveTimeout:
		atomic.AddUint32(&s.timeout, 1)
	}
}

// use bolt v1 to test keep alive
type testCase struct {
	KeepAlive *xprotocolKeepAlive
	Server    *mockServer
}

func newTestCase(t *testing.T, srvTimeout, keepTimeout time.Duration) *testCase {
	// start a mock server
	srv, err := newMockServer(srvTimeout)
	if err != nil {
		t.Fatal(err)
	}
	srv.GoServe()
	// make a connection to server
	info := &mockClusterInfo{
		name:  "test",
		limit: 1024,
	}
	cfg := v2.Host{
		HostConfig: v2.HostConfig{
			Address:    srv.AddrString(),
			TLSDisable: true, // ignore tls, for mock is nil
		},
	}
	host := cluster.NewSimpleHost(cfg, info)
	ctx := variable.NewVariableContext(context.Background())

	conn := host.CreateConnection(ctx)
	if err := conn.Connection.Connect(); err != nil {
		t.Fatalf("create conenction failed %v", err)
	}
	codec := str.NewStreamClient(ctx, bolt.ProtocolName, conn.Connection, host)
	if codec == nil {
		t.Fatal("codec is nil")
	}
	// start a keep alive
	keepAlive := NewKeepAlive(codec, (&bolt.XCodec{}).NewXProtocol(ctx), keepTimeout)
	keepAlive.StartIdleTimeout()
	return &testCase{
		KeepAlive: keepAlive.(*xprotocolKeepAlive),
		Server:    srv,
	}

}

func TestKeepAlive(t *testing.T) {
	tc := newTestCase(t, 0, time.Second)
	defer tc.Server.Close()
	testStats := &testStats{}
	tc.KeepAlive.AddCallback(testStats.Record)
	// test concurrency
	for i := 0; i < 5; i++ {
		go tc.KeepAlive.SendKeepAlive()
	}
	// wait response
	time.Sleep(2 * time.Second)
	if testStats.success != 5 {
		t.Error("keep alive handle success not enough", testStats)
	}
}

// when tick count more than 1, should send heart beat every tickCount intervals
func TestKeepAliveTickMore(t *testing.T) {
	tc := newTestCase(t, 0*time.Millisecond, 50*time.Millisecond)
	defer tc.Server.Close()
	defer RefreshKeepaliveConfig(DefaultKeepaliveConfig)

	RefreshKeepaliveConfig(KeepaliveConfig{
		TickCountIfFail:  1,
		TickCountIfSucc:  2,
		FailCountToClose: 6,
	})

	testStats := &testStats{}
	tc.KeepAlive.AddCallback(testStats.Record)

	// should tick 10 times, and success = 10 / tickCountIfSucc = 5
	for i := 0; i < 10; i++ {
		tc.KeepAlive.SendKeepAlive()
		time.Sleep(80 * time.Millisecond)
	}

	// wait all response
	time.Sleep(time.Second)
	assert.Equal(t, int(testStats.success), 10/2)
}

func TestKeepAliveTimeout(t *testing.T) {
	tc := newTestCase(t, 50*time.Millisecond, 10*time.Millisecond)
	defer tc.Server.Close()
	testStats := &testStats{}
	tc.KeepAlive.AddCallback(testStats.Record)
	// after 6 times, the connection will be closed and stop all keep alive action
	for i := 0; i < 10; i++ {
		tc.KeepAlive.SendKeepAlive()
		time.Sleep(80 * time.Millisecond)
	}
	// wait all response
	time.Sleep(time.Second)
	if testStats.timeout != 6 { // 6 is the max try times
		t.Error("keep alive handle failure not enough", testStats)
	}
}

func TestKeepAliveTimeoutAndSuccess(t *testing.T) {
	tc := newTestCase(t, 150*time.Millisecond, 20*time.Millisecond)
	defer tc.Server.Close()
	testStats := &testStats{}
	tc.KeepAlive.AddCallback(testStats.Record)
	// 5 times timeout, will not close the connection
	for i := 0; i < 5; i++ {
		tc.KeepAlive.SendKeepAlive()
		time.Sleep(200 * time.Millisecond)
	}
	// set no delay, will not timeout
	tc.Server.delay = 0
	tc.KeepAlive.SendKeepAlive()
	// wait response
	time.Sleep(time.Second)
	if testStats.success != 1 || testStats.timeout != 5 {
		t.Error("keep alive handle status not expected", testStats)
	}
	if tc.KeepAlive.heartbeatFailCount.Load() != 0 {
		t.Error("timeout count not reset by success")
	}

}

func TestKeepAliveIdleFree(t *testing.T) {
	// setup for test
	log.DefaultLogger.SetLogLevel(log.ERROR)
	maxIdleCount = 20
	// teardown for test
	defer func() {
		maxIdleCount = 0
		log.DefaultLogger.SetLogLevel(log.INFO)
	}()
	tc := newTestCase(t, 0, time.Second)
	defer tc.Server.Close()
	testStats := &testStats{}
	tc.KeepAlive.AddCallback(testStats.Record)

	var i uint32 = 0
	for ; i < maxIdleCount; i++ {
		tc.KeepAlive.SendKeepAlive()
		time.Sleep(10 * time.Millisecond)
	}
	// should be closed
	select {
	case <-tc.KeepAlive.stop:
		if testStats.timeout != 0 {
			t.Errorf("expected no timeout, but got: %d", testStats.timeout)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("expected close codec, but not, heartbeat: %v", testStats)
	}
}

func TestKeepAliveIdleFreeWithData(t *testing.T) {
	// setup for test
	log.DefaultLogger.SetLogLevel(log.ERROR)
	maxIdleCount = 40
	// teardown for test
	defer func() {
		maxIdleCount = 0
		log.DefaultLogger.SetLogLevel(log.INFO)
	}()
	tc := newTestCase(t, 0, time.Second)
	defer tc.Server.Close()
	ch := make(chan struct{})
	wg := sync.WaitGroup{}
	// 10ms a heartbeat, 400ms will send max count
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-ch:
				ticker.Stop()
				return
			case <-ticker.C:
				tc.KeepAlive.SendKeepAlive()
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(15 * time.Millisecond)
		for {
			select {
			case <-ch:
				ticker.Stop()
				return
			case <-ticker.C:
				// simulate a request stream
				tc.KeepAlive.Codec.NewStream(context.Background(), nil)
			}
		}
	}()
	select {
	case <-tc.KeepAlive.stop:
		t.Errorf("connection is closed")
	case <-time.After(2 * time.Second):
	}
	close(ch)
	wg.Wait()
}
