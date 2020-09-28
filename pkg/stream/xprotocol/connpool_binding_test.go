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
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/upstream/cluster"
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

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyConfigUpStreamProtocol, string(protocol.Xprotocol))
	ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, "dubbo")

	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	cl := basicCluster(addr, []string{addr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := &connpool{
		protocol: protocol.Xprotocol,
		tlsHash:  &types.HashValue{},
	}
	p.host.Store(host)

	var pool = NewPoolBinding(p)
	var pInst = pool.(*poolBinding)

	sConn, err := net.Dial("tcp4", addr)
	assert.Nil(t, err)

	var sstopChan = make(chan struct{})
	sConnI := network.NewServerConnection(context.Background(), sConn, sstopChan)

	ctx = mosnctx.WithValue(ctx, types.ContextKeyConnection, sConnI)
	ctx = mosnctx.WithValue(ctx, types.ContextKeyConnectionID, sConnI.ID())

	host, _, failReason := pInst.NewStream(ctx, nil)
	assert.Equal(t, failReason, types.PoolFailureReason(""))

	assert.Equal(t, len(pInst.idleClients[sConnI.ID()]), 1)
	// server stream conn close
	sConnI.Close(api.NoFlush, api.LocalClose)
	// should close the client stream conn
	assert.Equal(t, 0, len(pInst.idleClients[sConnI.ID()]))
}

func TestUpperClose(t *testing.T) {

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyConfigUpStreamProtocol, string(protocol.Xprotocol))
	ctx = mosnctx.WithValue(ctx, types.ContextSubProtocol, "dubbo")

	var addr = "127.0.0.1:10086"
	go server.start(t, addr)
	defer server.stop(t)
	// wait for server to start
	time.Sleep(time.Second * 2)

	cl := basicCluster(addr, []string{addr})
	host := cluster.NewSimpleHost(cl.Hosts[0], cluster.NewCluster(cl).Snapshot().ClusterInfo())

	p := &connpool{
		protocol: protocol.Xprotocol,
		tlsHash:  &types.HashValue{},
	}
	p.host.Store(host)

	var pool = NewPoolBinding(p)
	var pInst = pool.(*poolBinding)

	sConn, err := net.Dial("tcp4", addr)
	assert.Nil(t, err)

	var sstopChan = make(chan struct{})
	sConnI := network.NewServerConnection(context.Background(), sConn, sstopChan)

	ctx = mosnctx.WithValue(ctx, types.ContextKeyConnection, sConnI)
	ctx = mosnctx.WithValue(ctx, types.ContextKeyConnectionID, sConnI.ID())

	host, _, failReason := pInst.NewStream(ctx, nil)
	assert.Equal(t, failReason, types.PoolFailureReason(""))

	assert.Equal(t, len(pInst.idleClients[sConnI.ID()]), 1)

	// upstream close should close the downstream conn
	pInst.idleClients[sConnI.ID()][0].Close(errors.New("closeclose"))
	assert.Equal(t, len(pInst.idleClients[sConnI.ID()]), 0)
	assert.Equal(t, sConnI.State(), api.ConnClosed)
}
