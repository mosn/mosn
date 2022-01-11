package functiontest

import (
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/server"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

// Test Update stream filters by LDS can effect the connections that created before.
// we use some codes in faultinejct_test.go
// client->mosn->server
// protocol independent case
func TestUpdateStreamFilters(t *testing.T) {
	server.ResetAdapter()
	// start a server

	appAddr := "127.0.0.1:8080"
	server := util.NewRPCServer(t, appAddr, bolt.ProtocolName)
	server.GoServe()
	defer server.Close()
	// create mosn without stream filters
	clientMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateXProtocolProxyMesh(clientMeshAddr, []string{appAddr}, bolt.ProtocolName)
	mesh := mosn.NewMosn(cfg)
	mesh.Start()
	defer mesh.Close()
	time.Sleep(5 * time.Second)
	// send a request to mosn, create connection between mosns
	rpc, ok := server.(*util.RPCServer)
	if !ok {
		t.Fatal("not a expected rpc server")
	}
	clt := rpc.Client
	if err := clt.Connect(clientMeshAddr); err != nil {
		t.Fatalf("create connection to mosn failed, %v", err)
	}
	defer clt.Close()
	clt.SendRequestWithData("testdata")
	if !util.WaitMapEmpty(&clt.Waits, 2*time.Second) {
		t.Fatal("no expected response")
	}
	// add stream filters
	if err := updateListener(cfg, MakeFaultStr(500, 0)); err != nil {
		t.Fatalf("update listener failed, error: %v", err)
	}
	// set expected status
	clt.ExpectedStatus = int16(bolt.ResponseStatusUnknown)
	// send request to verify the stream filters is valid
	clt.SendRequestWithData("testdata")
	if !util.WaitMapEmpty(&clt.Waits, 2*time.Second) {
		t.Fatal("no expected response")
	}
	// update stream filters
	if err := updateListener(cfg, MakeFaultStr(200, 0)); err != nil {
		t.Fatalf("update listener failed, error: %v", err)
	}
	// verify stream fllters
	clt.ExpectedStatus = int16(bolt.ResponseStatusSuccess)
	clt.SendRequestWithData("testdata")
	if !util.WaitMapEmpty(&clt.Waits, 2*time.Second) {
		t.Fatal("no expected response")
	}

}

// call mosn LDS API
func updateListener(cfg *v2.MOSNConfig, faultstr string) error {
	// reset stream filters
	cfg.Servers[0].Listeners[0].StreamFilters = cfg.Servers[0].Listeners[0].StreamFilters[:0]
	AddFaultInject(cfg, "proxyListener", faultstr)
	// get config
	lc := cfg.Servers[0].Listeners[0]
	return server.GetListenerAdapterInstance().AddOrUpdateListener("", &lc)
}
