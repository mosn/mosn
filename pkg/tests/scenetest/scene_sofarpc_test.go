package tests
//
//import (
//	"testing"
//	"time"
//
//	"github.com/orcaman/concurrent-map"
//	"github.com/alipay/sofamosn/pkg/mosn"
//	"github.com/alipay/sofamosn/pkg/protocol"
//)
//
//func TestSofaRpc(t *testing.T) {
//	sofaAddr := "127.0.0.1:8080"
//	meshAddr := "127.0.0.1:2045"
//	server := NewUpstreamServer(t, sofaAddr, ServeBoltV1)
//	server.GoServe()
//	defer server.Close()
//	mesh_config := CreateSimpleMeshConfig(meshAddr, []string{sofaAddr}, protocol.SofaRpc, protocol.SofaRpc)
//	go mosn.Start(mesh_config, "", "")
//	time.Sleep(5 * time.Second) //wait mesh and server start
//	//client
//	client := &BoltV1Client{
//		t:        t,
//		ClientId: "testClient",
//		Waits:    cmap.New(),
//	}
//	client.Connect(meshAddr)
//	for i := 0; i < 20; i++ {
//		client.SendRequest()
//	}
//	<-time.After(10 * time.Second)
//	if !client.Waits.IsEmpty() {
//		t.Errorf("exists request no response\n")
//	}
//}
