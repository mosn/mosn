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

package msgconnpool

import (
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"

	"github.com/stretchr/testify/assert"
)

// myFilter implements 2 interfaces
//  1. KeepAlive -> GetKeepAliveData + Stop + SaveHeartBeatFailCallback
//  2. ReadFilter -> OnData
type myFilter struct {
	keepaliveFrameTimeout time.Duration
	heartbeatTriggerCount int
	failCallback          atomic.Value // func()
}

// ~ pool KeepAlive
func (mf *myFilter) GetKeepAliveData() []byte {
	mf.heartbeatTriggerCount++

	time.AfterFunc(mf.keepaliveFrameTimeout, func() {
		mf.heartbeatFrameFail()
	})

	return []byte("keepalive data")
}

// ~ pool KeepAlive
func (mf *myFilter) Stop() {
	mf.failCallback.Store(func() {})
	// and stop all timers if any
}

// ~ pool KeepAlive
func (mf *myFilter) SaveHeartBeatFailCallback(callback func()) {
	mf.failCallback.Store(callback)
}

// in OnData, after received keepalive packet
func (mf *myFilter) heartbeatFrameSuccess(frameID int) {
	// remove this frame from your stream map
}

func (mf *myFilter) heartbeatFrameFail() {
	v := mf.failCallback.Load()
	if callback, ok := v.(func()); ok && callback != nil {
		callback()
	}
}

func (mf *myFilter) OnData(ioBuffer buffer.IoBuffer) api.FilterStatus {
	// read from ioBuffer
	// if frame.isHeartbeat {
	//   1. decode from frame
	frameID := 1
	//   2. find keepalive object from myFilter and delete keepalive callback
	mf.heartbeatFrameSuccess(frameID)
	// if response error
	// mf.mkp.heartbeatFrameFail()
	//   3. done
	fmt.Println("OnData from server", ioBuffer)
	return api.Stop
}

func (mf *myFilter) OnNewConnection() api.FilterStatus {
	return api.Continue
}

func (mf *myFilter) InitializeReadFilterCallbacks(callbacks api.ReadFilterCallbacks) {}

func TestExampleHeartBeatSuccess(t *testing.T) {
	// setup
	oldDefaultConnReadTimeout := types.DefaultConnReadTimeout
	types.DefaultConnReadTimeout = time.Second * 2
	// tear down
	defer func() {
		types.DefaultConnReadTimeout = oldDefaultConnReadTimeout
	}()

	server := tcpServer{
		addr: fmt.Sprint("127.0.0.1:10001"),
	}

	server.startServer(false, true)
	defer server.stop()

	// keepalive object is connection level
	// every connection should has a separate keepalive object
	var previousKeepalive *myFilter
	c := NewConn(server.addr, -1,
		func() ([]api.ReadFilter, KeepAlive) {
			var mf = &myFilter{
				keepaliveFrameTimeout: time.Hour, // never timeout
			}

			previousKeepalive = mf

			return []api.ReadFilter{mf}, mf
		}, true)
	defer c.Destroy()

	// ensure the heartbeat is triggered by ReadTimeout
	time.Sleep(types.DefaultConnReadTimeout + time.Second*5)

	assert.Less(t, 0, previousKeepalive.heartbeatTriggerCount)
	fmt.Println("heart beat trigger count", previousKeepalive.heartbeatTriggerCount)
}

func TestHeartBeatTimeoutFail(t *testing.T) {
	// setup
	oldDefaultConnReadTimeout := types.DefaultConnReadTimeout
	types.DefaultConnReadTimeout = time.Second * 2
	// tear down
	defer func() {
		types.DefaultConnReadTimeout = oldDefaultConnReadTimeout
	}()

	server := tcpServer{
		addr: fmt.Sprint("127.0.0.1:10001"),
	}

	server.startServer(false, false)
	defer server.stop()

	var previousKeepalive *myFilter
	c := NewConn(server.addr, -1,
		func() ([]api.ReadFilter, KeepAlive) {
			mf := &myFilter{
				keepaliveFrameTimeout: types.DefaultConnReadTimeout - time.Second,
			}
			previousKeepalive = mf
			return nil, mf
		}, true)
	defer c.Destroy()

	// ensure the heart beat is triggered
	time.Sleep(types.DefaultConnReadTimeout + time.Second*5)

	assert.Less(t, 0, previousKeepalive.heartbeatTriggerCount)
}

func TestReconnectTimesLimit(t *testing.T) {
	invalidAddr := "127.0.0.1:12345"
	tryTimes := 2
	pool := NewConn(invalidAddr, tryTimes, nil, true)
	defer pool.Destroy()
	time.Sleep(time.Second * 10)
	poolReal := pool.(*connpool)
	fmt.Println("retry for", poolReal.connTryTimes, "times")

	assert.LessOrEqual(t, poolReal.connTryTimes, tryTimes)
}

func TestAutoReconnectAfterRemoteClose(t *testing.T) {
	server := tcpServer{
		addr: fmt.Sprint("127.0.0.1:10001"),
	}

	pool := NewConn(server.addr, 100000, nil, true)
	defer pool.Destroy()
	server.startServer(true, false)
	defer server.stop()

	time.Sleep(time.Second * 5)
	// the connection should be connected
	assert.Equal(t, pool.State(), Available)
}

type tcpServer struct {
	listener net.Listener
	addr     string
	started  uint64

	conns []net.Conn
}

func (s *tcpServer) stop() {
	s.listener.Close()
	for _, c := range s.conns {
		c.Close()
	}

	s.conns = nil
}

func (s *tcpServer) startServer(rejectFirstConn bool, writeData bool) {
	var err error

	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		fmt.Println("listen failed", err)
	}

	go func() {
		for i := 0; ; i++ {
			c, e := s.listener.Accept()
			if e != nil {
				break
			}

			if i == 0 && rejectFirstConn {
				c.Close()
				continue
			}

			s.conns = append(s.conns, c)
			go func() {
				var buf = make([]byte, 1024)
				for {
					n, e := c.Read(buf)
					if e != nil {
						break
					}

					buf = buf[:n]
					if writeData {
						c.Write([]byte("1"))
					}
				}
			}()
		}
	}()
}
