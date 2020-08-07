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
	"net"
	"testing"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
)

type mockEventListener struct {
	recvStatus bool
	stopChan   chan struct{}
}

func (e *mockEventListener) GetRecvStatus() bool {
	return e.recvStatus
}

func (e *mockEventListener) SetRecvStatus(state bool) {
	e.recvStatus = state
}

func (e *mockEventListener) OnAccept(rawc net.Conn, useOriginalDst bool, oriRemoteAddr net.Addr, c chan api.Connection, buf []byte) {
	e.recvStatus = true
	rawc.Close()
}

func (e *mockEventListener) OnNewConnection(ctx context.Context, conn api.Connection) {
}

func (e *mockEventListener) OnClose() {}

func (e *mockEventListener) PreStopHook(ctx context.Context) func() error {
	return nil
}

func TestListenerStart(t *testing.T) {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:10101")
	cfg := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "test_listener",
			BindToPort: true,
		},
		PerConnBufferLimitBytes: 1024,
		Addr:                    addr,
	}
	ln := NewListener(cfg)
	el := &mockEventListener{}
	ln.SetListenerCallbacks(el)
	go ln.Start(nil, false) // start
	time.Sleep(3 * time.Second)
	check := func(t *testing.T) bool {
		conn, err := net.Dial("tcp", "127.0.0.1:10101")
		if err != nil {
			t.Logf("dial error: %v", err)
			return false
		}
		defer conn.Close()
		return true
	}
	if !check(t) {
		t.Error("listener start check failed")
	}
	// duplicate start, will be ignored, return directly
	for i := 0; i < 10; i++ {
		ch := make(chan struct{})
		go func() {
			ln.Start(nil, false)
			close(ch)
		}()
		select {
		case <-ch:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("start not be ignored")
		}
	}
	// close listener
	if err := ln.Close(nil); err != nil {
		t.Errorf("Close listener failed, %v", err)
	}
	time.Sleep(3 * time.Second)
	if check(t) {
		t.Error("listener closed, but still can be dial success")
	}
	// start, but not restart, will be failed
	go ln.Start(nil, false)
	time.Sleep(time.Second)
	if check(t) {
		t.Error("listener start")
	}
	// restart
	go ln.Start(nil, true)
	time.Sleep(time.Second)
	if !check(t) {
		t.Error("listener restart check failed")
	}

}

func TestUDPListenerStart(t *testing.T) {
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:10101")
	cfg := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       "test_listener",
			BindToPort: true,
			Network:    "udp",
		},
		PerConnBufferLimitBytes: 1024,
		Addr:                    addr,
	}
	ln := NewListener(cfg)
	el := &mockEventListener{}
	ln.SetListenerCallbacks(el)
	go ln.Start(nil, false) // start
	time.Sleep(time.Second)

	conn, _ := net.DialUDP("udp", nil, addr)
	defer conn.Close()
	check := func(t *testing.T) bool {
		conn.Write([]byte("test"))
		time.Sleep(time.Second)
		res := el.GetRecvStatus()
		el.SetRecvStatus(false)
		return res
	}
	if !check(t) {
		t.Errorf("udp listen accept failed")
	}

	// duplicate start, will be ignored, return directly
	for i := 0; i < 10; i++ {
		ch := make(chan struct{})
		go func() {
			ln.Start(nil, false)
			close(ch)
		}()
		select {
		case <-ch:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("start not be ignored")
		}
	}

	// close listener
	if err := ln.Close(nil); err != nil {
		t.Errorf("Close listener failed, %v", err)
	}
	time.Sleep(2*time.Second)
	if check(t) {
		t.Error("listener closed, but still can be dial success")
	}

	// start, but not restart, will be failed
	go ln.Start(nil, false)
	time.Sleep(2*time.Second)
	if check(t) {
		t.Error("listener start")
	}
}
