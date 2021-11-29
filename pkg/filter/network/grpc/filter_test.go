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

package grpc

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"mosn.io/api"
)

func TestFilterOnNewConnection(t *testing.T) {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	ln := &Listener{
		accepts: make(chan net.Conn), // mock a listener without buffer,
		addr:    addr,
	}
	f := NewGrpcFilter(nil, ln)
	verify := func(expected api.FilterStatus) {
		mockCb := &MockReadFilterCallbacks{
			conn: &MockConnection{},
		}
		f.InitializeReadFilterCallbacks(mockCb)
		timer := time.NewTimer(5 * time.Second)

		ch := make(chan struct{})
		go func() {
			status := f.OnNewConnection()
			// no listener accept, read timeout
			require.Equal(t, expected, status)
			if expected == api.Stop {
				require.True(t, mockCb.conn.closed)
			} else {
				require.False(t, mockCb.conn.closed)
			}
			close(ch)
		}()
		select {
		case <-ch:
			timer.Stop()
		case <-timer.C:
			t.Fatalf("OnNewConnection without returns")
		}

	}
	verify(api.Stop)
	// start a accept
	go func() {
		for {
			_, err := ln.Accept()
			if err != nil {
				return
			}
		}
	}()
	verify(api.Continue)
	// stop listener
	ln.Close()
	verify(api.Stop)
}
