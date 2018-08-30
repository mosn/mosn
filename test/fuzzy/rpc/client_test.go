package rpc

import (
	"fmt"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/fuzzy"
	"github.com/alipay/sofa-mosn/test/util"
)

func runClientClose(t *testing.T, stop chan struct{}, meshAddr string) {
	var clients []*RPCStatusClient
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("client#%d", i)
		client := NewRPCClient(t, id, util.Bolt1, meshAddr)
		if err := client.Connect(); err != nil {
			t.Fatal(err)
		}
		fuzzy.FuzzyClient(stop, client)
		clients = append(clients, client)
	}
	//client
	client := NewRPCClient(t, "randomclient", util.Bolt1, meshAddr)
	client.RandomEvent(stop)
	fuzzy.FuzzyClient(stop, client)
	<-time.After(caseDuration)
	close(stop)
	time.Sleep(5 * time.Second)
	log.StartLogger.Infof("[FUZZY TEST] client #%s success:%d, failure: %d", client.ClientID, client.successCount, client.failureCount)
	for i, c := range clients {
		if c.failureCount != 0 {
			t.Errorf("case%d #%d have failure request: %d\n", caseIndex, i, c.failureCount)
		}
		if c.unexpectedCount != 0 {
			t.Errorf("case%d #%d have unexpected request: %d\n", caseIndex, i, c.failureCount)
		}
		log.StartLogger.Infof("[FUZZY TEST] #%d request count: %d\n", i, c.successCount)
	}
}

func TestClientCloseProxy(t *testing.T) {
	caseIndex++
	log.StartLogger.Infof("[FUZZY TEST] RPC ClientClose ProxyMode %d", caseIndex)
	serverList := []string{"127.0.0.1:8081"}
	stopClient := make(chan struct{})
	stopServer := make(chan struct{})
	meshAddr := fuzzy.CreateMeshProxy(t, stopServer, serverList, protocol.SofaRPC)
	CreateServers(t, serverList, stopServer)
	runClientClose(t, stopClient, meshAddr)
	close(stopServer)
	// wait server close
	time.Sleep(time.Second)

}

func runClientCloseMeshToMesh(t *testing.T, proto types.Protocol) {
	serverList := []string{"127.0.0.1:8081"}
	stopClient := make(chan struct{})
	stopServer := make(chan struct{})
	meshAddr := fuzzy.CreateMeshCluster(t, stopServer, serverList, protocol.SofaRPC, proto)
	CreateServers(t, serverList, stopServer)
	runClientClose(t, stopClient, meshAddr)
	close(stopServer)
	// wait server close
	time.Sleep(time.Second)

}
func TestClientCloseToHTTP1(t *testing.T) {
	caseIndex++
	log.StartLogger.Infof("[FUZZY TEST] RPC ClientClose HTTP1 %d", caseIndex)
	runClientCloseMeshToMesh(t, protocol.HTTP1)
}
func TestClientCloseToHTTP2(t *testing.T) {
	caseIndex++
	log.StartLogger.Infof("[FUZZY TEST] RPC ClientClose HTTP2 %d", caseIndex)
	runClientCloseMeshToMesh(t, protocol.HTTP2)
}
func TestClientCloseToSofaRPC(t *testing.T) {
	caseIndex++
	log.StartLogger.Infof("[FUZZY TEST] RPC ClientClose SofaRPC %d", caseIndex)
	runClientCloseMeshToMesh(t, protocol.SofaRPC)
}
