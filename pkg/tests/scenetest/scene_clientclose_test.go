package tests

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/mosn"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//one client close should not effect others
func TestClientClose(t *testing.T) {
	sofaAddr := "127.0.0.1:8080"
	meshAddr := "127.0.0.1:2045"
	server := NewUpstreamServer(t, sofaAddr, ServeBoltV1)
	server.GoServe()
	defer server.Close()
	mesh_config := CreateSimpleMeshConfig(meshAddr, []string{sofaAddr}, protocol.SofaRpc, protocol.SofaRpc)
	go mosn.Start(mesh_config, "", "")
	time.Sleep(5 * time.Second) //wait mesh and server start
	var stopChans []chan struct{}
	//send request with cancel
	wg := sync.WaitGroup{}

	call := func(client *BoltV1Client, stop chan struct{}) {
		defer wg.Done()
		for {
			select {
			case <-stop:
				//before close, should check all request get reponse
				<-time.After(time.Second)
				if !client.Waits.IsEmpty() {
					t.Errorf("client %s has request timeout\n", client.ClientId)
				}
				client.conn.Close(types.NoFlush, types.LocalClose)
				client.Stats()
				return
			default:
				client.SendRequest()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	makeClient := func(clientId string, stop chan struct{}) {
		client := &BoltV1Client{
			t:        t,
			ClientId: clientId,
			Waits:    cmap.New(),
		}
		if err := client.Connect(meshAddr); err != nil {
			t.Fatalf("client %s connect to mesh failed, error: %v\n", clientId, err)
		}
		wg.Add(1)
		go call(client, stop)
	}
	//create 2 client
	for i := 0; i < 2; i++ {
		stop := make(chan struct{})
		stopChans = append(stopChans, stop)
		makeClient(fmt.Sprintf("client.%d", i), stop)
	}
	//create a client will be closed
	stop := make(chan struct{})
	makeClient("client.close", stop)
	//close the client randomly, 1s ~ 3s
	closetime := time.Duration(rand.Intn(2000)) * time.Millisecond
	<-time.After(time.Second + closetime)
	close(stop)
	//create a new client
	<-time.After(3 * time.Second)
	ch := make(chan struct{})
	stopChans = append(stopChans, ch)
	makeClient("client.new", ch)
	//close all client
	<-time.After(5 * time.Second)
	for _, cancel := range stopChans {
		close(cancel)
	}
	//wait close and verify with timeout
	ok := make(chan struct{})
	go func() {
		wg.Wait()
		close(ok)
	}()
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("test timeout\n")
	case <-ok:
	}
}
