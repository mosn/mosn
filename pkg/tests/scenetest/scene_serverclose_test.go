package tests

import (
	"testing"
	"time"

	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/mosn"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
)

//when a upstream server has been closed
//the client should get a error response
func TestServerClose(t *testing.T) {
	meshAddr := "127.0.0.1:2045"
	serverAddrs := []string{
		"127.0.0.1:8080",
		"127.0.0.1:8081",
	}
	servers := []*UpstreamServer{}
	for _, addr := range serverAddrs {
		server := NewUpstreamServer(t, addr, ServeBoltV1)
		server.GoServe()
		defer server.Close()
		servers = append(servers, server)
	}
	mesh_config := CreateSimpleMeshConfig(meshAddr, serverAddrs, protocol.SofaRpc, protocol.SofaRpc)
	go mosn.Start(mesh_config, "", "")
	time.Sleep(5 * time.Second) //wait mesh and server start
	client := &BoltV1Client{
		t:        t,
		ClientId: "testClient",
		Waits:    cmap.New(),
	}
	client.Connect(meshAddr)
	//send request
	go func() {
		for i := 0; i < 10; i++ {
			client.SendRequest()
			time.Sleep(time.Second)
		}
	}()
	//close a server after 4 seconds
	go func() {
		<-time.After(4 * time.Second)
		servers[0].Close()
	}()
	<-time.After(15 * time.Second) //wait request finish
	if !client.Waits.IsEmpty() {
		t.Errorf("some request get no response\n")
	}
}
