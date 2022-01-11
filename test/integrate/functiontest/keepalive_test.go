package functiontest

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"mosn.io/api"

	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

type heartBeatServer struct {
	util.UpstreamServer
	HeartBeatCount uint32
	boltProto      api.XProtocol
}

func (s *heartBeatServer) ServeBoltOrHeartbeat(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		cmd, _ := s.boltProto.Decode(nil, iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*bolt.Request); ok {
			var iobufresp types.IoBuffer
			var err error
			switch req.CmdCode {
			case bolt.CmdCodeHeartbeat:
				hbAck := s.boltProto.Reply(context.TODO(), req)
				iobufresp, err = s.boltProto.Encode(context.Background(), hbAck)
				atomic.AddUint32(&s.HeartBeatCount, 1)
			case bolt.CmdCodeRpcRequest:
				resp := bolt.NewRpcResponse(req.RequestId, bolt.ResponseStatusSuccess, nil, nil)
				iobufresp, err = s.boltProto.Encode(context.Background(), resp)
			}
			if err != nil {
				return nil, true
			}
			return iobufresp.Bytes(), true
		}
		return nil, true
	}
	util.ServeRPC(t, conn, response)
}

// Test Proxy Mode
// TODO: support protocol convert
func TestKeepAlive(t *testing.T) {
	appAddr := "127.0.0.1:8080"
	server := &heartBeatServer{}
	server.UpstreamServer = util.NewUpstreamServer(t, appAddr, server.ServeBoltOrHeartbeat)
	server.boltProto = (&bolt.XCodec{}).NewXProtocol(context.Background())
	server.GoServe()
	clientMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateXProtocolProxyMesh(clientMeshAddr, []string{appAddr}, bolt.ProtocolName)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	stop := make(chan bool)
	go func() {
		<-stop
		server.Close()
		mesh.Close()
		stop <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
	// start case
	client := util.NewRPCClient(t, "testKeepAlive", bolt.ProtocolName)
	if err := client.Connect(clientMeshAddr); err != nil {
		t.Fatal(err)
	}
	// send request, make a connection
	client.SendRequest()
	// sleep, makes the conn idle, mosn will keep alive with upstream
	// interval 15s, sleep to wait 2 heart beat
	time.Sleep(2*types.DefaultConnReadTimeout + 3*time.Second)
	// send request interval, to stop keep avlie
	st := make(chan bool)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-st:
				ticker.Stop()
				st <- true
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
	// stop the ticker goroutine and then stop the case
	st <- true
	<-st
	// stop the case
	stop <- true
	<-stop
}
