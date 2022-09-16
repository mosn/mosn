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
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
	"mosn.io/pkg/variable"
)

type serverType struct {
	listener net.Listener
	doneChan chan struct{}
}

func (s *serverType) start(t *testing.T, addr string) {
	s.doneChan = make(chan struct{})
	var err error
	s.listener, err = net.Listen("tcp4", addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	assert.Nil(t, err)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			break
		}
		go func() {
		readLoop:
			for {
				conn.SetReadDeadline(time.Now().Add(time.Second * 15))
				var buf = make([]byte, 1024)
				conn.Read(buf)
				select {
				case <-s.doneChan:
					return
				default:
					continue readLoop
				}
			}
		}()
	}
}

func (s *serverType) stop(t *testing.T) {
	s.listener.Close()
	close(s.doneChan)
}

var server = serverType{}

// the upper close should close the down
// the down close should close the upper
func TestBinding(t *testing.T) {
	TestDownClose(t)
	TestUpperClose(t)
}

func TestDownClose(t *testing.T) {

	ctx := variable.NewVariableContext(context.Background())

	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	cl := basicCluster(addr, []string{addr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := &connpool{
		protocol: api.ProtocolName(dubbo.ProtocolName),
		tlsHash:  &types.HashValue{},
		codec:    &dubbo.XCodec{},
	}
	p.host.Store(host)

	var pool = NewPoolBinding(p)
	var pInst = pool.(*poolBinding)

	sConn, err := net.Dial("tcp4", addr)
	assert.Nil(t, err)

	var sstopChan = make(chan struct{})
	sConnI := network.NewServerConnection(context.Background(), sConn, sstopChan)

	_ = variable.Set(ctx, types.VariableConnection, sConnI)
	_ = variable.Set(ctx, types.VariableConnectionID, sConnI.ID())

	host, _, failReason := pInst.NewStream(ctx, nil)
	assert.Equal(t, failReason, types.PoolFailureReason(""))

	assert.NotNil(t, pInst.idleClients[sConnI.ID()])
	// server stream conn close
	sConnI.Close(api.NoFlush, api.LocalClose)
	// should close the client stream conn
	assert.Nil(t, pInst.idleClients[sConnI.ID()])

	// close of pool should not panic
	pInst.Close()
}

func TestUpperClose(t *testing.T) {

	ctx := variable.NewVariableContext(context.Background())

	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	cl := basicCluster(addr, []string{addr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := &connpool{
		protocol: api.ProtocolName(dubbo.ProtocolName),
		tlsHash:  &types.HashValue{},
		codec:    &dubbo.XCodec{},
	}
	p.host.Store(host)

	var pool = NewPoolBinding(p)
	var pInst = pool.(*poolBinding)

	sConn, err := net.Dial("tcp4", addr)
	assert.Nil(t, err)

	var sstopChan = make(chan struct{})
	sConnI := network.NewServerConnection(context.Background(), sConn, sstopChan)

	_ = variable.Set(ctx, types.VariableConnection, sConnI)
	_ = variable.Set(ctx, types.VariableConnectionID, sConnI.ID())

	host, _, failReason := pInst.NewStream(ctx, nil)
	assert.Equal(t, failReason, types.PoolFailureReason(""))

	assert.NotNil(t, pInst.idleClients[sConnI.ID()])

	// upstream close should close the downstream conn
	pInst.idleClients[sConnI.ID()].Close(errors.New("closeclose"))
	assert.Nil(t, pInst.idleClients[sConnI.ID()])
	assert.Equal(t, sConnI.State(), api.ConnClosed)

	// client has already closed
	// close the connpool should not panic
	pInst.Close()
}

func TestUpperGoAway(t *testing.T) {

	ctx := variable.NewVariableContext(context.Background())

	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	cl := basicCluster(addr, []string{addr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := &connpool{
		protocol: api.ProtocolName(dubbo.ProtocolName),
		tlsHash:  &types.HashValue{},
		codec:    &dubbo.XCodec{},
	}
	p.host.Store(host)

	var pool = NewPoolBinding(p)
	var pInst = pool.(*poolBinding)

	sConn, err := net.Dial("tcp4", addr)
	assert.Nil(t, err)

	var sstopChan = make(chan struct{})
	sConnI := network.NewServerConnection(context.Background(), sConn, sstopChan)

	_ = variable.Set(ctx, types.VarConnection, sConnI)
	_ = variable.Set(ctx, types.VarConnectionID, sConnI.ID())

	host, _, failReason := pInst.NewStream(ctx, nil)
	assert.Equal(t, failReason, types.PoolFailureReason(""))
	assert.NotNil(t, pInst.idleClients[sConnI.ID()])

	cb1 := pInst.idleClients[sConnI.ID()]

	// upstream goaway and downstream should be active
	pInst.idleClients[sConnI.ID()].OnGoAway()
	assert.Nil(t, pInst.idleClients[sConnI.ID()])
	assert.Equal(t, sConnI.State(), api.ConnActive)

	// after goaway , upstream close will not affect the downstream
	cb1.Close(errors.New("closeclose"))
	assert.Equal(t, sConnI.State(), api.ConnActive)

	// after goaway , we can choose another upstream
	ctx = variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VarConnection, sConnI)
	_ = variable.Set(ctx, types.VarConnectionID, sConnI.ID())

	host, _, failReason = pInst.NewStream(ctx, nil)
	assert.Equal(t, failReason, types.PoolFailureReason(""))
	assert.NotNil(t, pInst.idleClients[sConnI.ID()])
	cb2 := pInst.idleClients[sConnI.ID()]

	assert.NotEqual(t, cb1.host.Connection.ID(), cb2.host.Connection.ID())

	// upstream close should close the downstream conn
	pInst.idleClients[sConnI.ID()].Close(errors.New("closeclose"))
	assert.Nil(t, pInst.idleClients[sConnI.ID()])
	assert.Equal(t, sConnI.State(), api.ConnClosed)

	// client has already closed
	// close the connpool should not panic
	pInst.Close()
}
