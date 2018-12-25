package functiontest

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
)

type heartBeatServer struct {
	util.UpstreamServer
	HeartBeatCount uint32
}

func (s *heartBeatServer) ServeBoltOrHeartbeat(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		cmd, _ := codec.BoltCodec.Decode(nil, iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*sofarpc.BoltRequest); ok {
			var iobufresp types.IoBuffer
			var err error
			switch req.CommandCode() {
			case sofarpc.HEARTBEAT:
				hbAck := sofarpc.NewHeartbeatAck(req.ProtocolCode())
				hbAck.SetRequestID(req.RequestID())
				iobufresp, err = codec.BoltCodec.Encode(context.Background(), hbAck)
				atomic.AddUint32(&s.HeartBeatCount, 1)
			case sofarpc.RPC_REQUEST:
				resp := util.BuildBoltV1Response(req)
				iobufresp, err = codec.BoltCodec.Encode(nil, resp)
			}
			if err != nil {
				return nil, true
			}
			return iobufresp.Bytes(), true
		}
		return nil, true
	}
	util.ServeSofaRPC(t, conn, response)
}

// Test Proxy Mode
// TODO: support protocol convert
func TestKeepAlive(t *testing.T) {
	appAddr := "127.0.0.1:8080"
	server := &heartBeatServer{}
	server.UpstreamServer = util.NewUpstreamServer(t, appAddr, server.ServeBoltOrHeartbeat)
	server.GoServe()
	clientMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateProxyMesh(clientMeshAddr, []string{appAddr}, protocol.SofaRPC)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	stop := make(chan struct{})
	go func() {
		<-stop
		server.Close()
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
	// start case
	client := util.NewRPCClient(t, "testKeepAlive", util.Bolt1)
	if err := client.Connect(clientMeshAddr); err != nil {
		t.Fatal(err)
	}
	// send request, make a connection
	client.SendRequest()
	// sleep, makes the conn idle, mosn will keep alive with upstream
	// interval 15s, sleep to wait 2 heart beat
	time.Sleep(2*types.DefaultConnReadTimeout + 3*time.Second)
	// send request interval, to stop keep avlie
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				client.SendRequest()
			}
		}
	}()
	time.Sleep(types.DefaultConnReadTimeout)
	// check, should have and only have 2 heart beat
	if server.HeartBeatCount != 2 {
		t.Errorf("server receive %d heart beats", server.HeartBeatCount)
	}
	// stop the case
	close(stop)
}
