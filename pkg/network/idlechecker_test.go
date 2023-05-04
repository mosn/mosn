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

package network

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

// types.ListenerEventListener
type mockHandler struct {
	stopChan chan struct{}
}

func (h *mockHandler) OnAccept(rawc net.Conn, useOriginalDst bool, oriRemoteAddr net.Addr, c chan api.Connection, buf []byte, listeners []api.ConnectionEventListener) {
	ctx := context.Background()
	conn := NewServerConnection(ctx, rawc, h.stopChan)
	conn.SetIdleTimeout(types.DefaultConnReadTimeout, 3*time.Second)
	h.OnNewConnection(ctx, conn)
}

func (h *mockHandler) OnNewConnection(ctx context.Context, conn api.Connection) {
	conn.Start(ctx)
}

func (h *mockHandler) OnShutdown() {
}

func (h *mockHandler) OnClose() {
}

func (h *mockHandler) PreStopHook(ctx context.Context) func() error {
	return nil
}

const testAddress = "127.0.0.1:18080"

func _createListener(address string) types.Listener {
	addr, _ := net.ResolveTCPAddr("tcp", address)
	lc := &v2.Listener{
		Addr:                    addr,
		PerConnBufferLimitBytes: 1 << 15,
		ListenerConfig: v2.ListenerConfig{
			BindToPort: true,
		},
	}
	return GetListenerFactory()(lc)
}

func TestIdleChecker(t *testing.T) {
	// setup
	oldDefaultConnReadTimeout := types.DefaultConnReadTimeout
	types.DefaultConnReadTimeout = time.Second
	// tear down
	defer func() {
		types.DefaultConnReadTimeout = oldDefaultConnReadTimeout
	}()
	ln := _createListener(testAddress)
	defer func() {
		ln.Close(nil)
		time.Sleep(time.Second) // wait listener really closed
	}()
	ln.SetListenerCallbacks(&mockHandler{
		stopChan: make(chan struct{}),
	})
	go ln.Start(context.Background(), false)
	time.Sleep(2 * time.Second)
	// create a connection, send nothing, will be closed after a while
	start := time.Now()
	conn, err := net.Dial("tcp", testAddress)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	ch := make(chan error)
	go func() {
		buf := make([]byte, 100)
		_, err := conn.Read(buf)
		ch <- err
	}()
	select {
	case err := <-ch:
		if err != io.EOF {
			t.Fatal("expected a closed connection error, but got: ", err)
		}
		duration := time.Now().Sub(start)
		if duration < time.Duration(3)*types.DefaultConnReadTimeout ||
			duration > time.Duration(4)*types.DefaultConnReadTimeout {
			t.Fatalf("expected close connection when idle max, but close at %v", duration)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("listener did not close the connection")
	}
}

func TestIdleCheckerWithData(t *testing.T) {
	// setup
	oldDefaultConnReadTimeout := types.DefaultConnReadTimeout
	types.DefaultConnReadTimeout = time.Second
	// tear down
	defer func() {
		types.DefaultConnReadTimeout = oldDefaultConnReadTimeout
	}()

	ln := _createListener(testAddress)
	defer func() {
		ln.Close(nil)
		time.Sleep(time.Second) // wait listener really closed
	}()
	ln.SetListenerCallbacks(&mockHandler{
		stopChan: make(chan struct{}),
	})
	go ln.Start(context.Background(), false)
	time.Sleep(2 * time.Second)
	conn, err := net.Dial("tcp", testAddress)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	// 1s trigger a read timeout, needs 3 times to close connection
	// 2s send a data, clean the counter, never close the connection
	// no data response, conn.Read never get data
	go func() {
		ticker := time.NewTicker(2 * types.DefaultConnReadTimeout)
		for _ = range ticker.C {
			conn.Write([]byte{0x01})
		}
	}()
	ch := make(chan error)
	go func() {
		buf := make([]byte, 100)
		_, err := conn.Read(buf)
		ch <- err
	}()
	// should not be closed
	select {
	case <-ch:
		t.Fatal("connection read data or error, but expected not")
	case <-time.After(10 * time.Second):
		conn.Close()
	}

}

func TestGetIdleCount(t *testing.T) {
	// teardown
	if maxIdleCount := getIdleCount(types.DefaultConnReadTimeout, 100*time.Second); maxIdleCount != 7 {
		t.Error("set idle timeout unexpected:", maxIdleCount)
	}
	if maxIdleCount := getIdleCount(types.DefaultConnReadTimeout, 90*time.Second); maxIdleCount != 6 {
		t.Error("set idle timeout unexpected:", maxIdleCount)
	}
	if maxIdleCount := getIdleCount(types.DefaultConnReadTimeout, 0); maxIdleCount != 0 {
		t.Error("set idle timeout unexpected:", maxIdleCount)
	}
}
