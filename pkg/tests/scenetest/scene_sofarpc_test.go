package tests

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/mosn"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//types.StreamReceiver
type sofaClient struct {
	t               *testing.T
	ClientId        string
	CurrentStreamId uint32
	Codec           stream.CodecClient
	Waits           cmap.ConcurrentMap
}

func (c *sofaClient) Connect(addr string) {
	stopChan := make(chan bool)
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	if err := cc.Connect(true); err != nil {
		c.t.Errorf("client[%s] connect to server error: %v\n", c.ClientId, err)
		return
	}
	c.Codec = stream.NewCodecClient(nil, protocol.SofaRpc, cc, nil)
}
func (c *sofaClient) SendRequest() {
	id := atomic.AddUint32(&c.CurrentStreamId, 1)
	streamId := sofarpc.StreamIDConvert(id)
	requestEncoder := c.Codec.NewStream(streamId, c)
	headers := buildBoltV1Request(id)
	requestEncoder.AppendHeaders(headers, true)
	c.Waits.Set(streamId, streamId)
}

//
func (c *sofaClient) OnReceiveData(data types.IoBuffer, endStream bool) {
}
func (c *sofaClient) OnReceiveTrailers(trailers map[string]string) {
}
func (c *sofaClient) OnDecodeError(err error, headers map[string]string) {
}
func (c *sofaClient) OnReceiveHeaders(headers map[string]string, endStream bool) {
	streamId, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]
	if ok {
		if _, ok := c.Waits.Get(streamId); ok {
			//c.t.Logf("Get Stream Response: %s ,headers: %v\n", streamId, headers)
			c.Waits.Remove(streamId)
		}
	}
}

func TestSofaRpc(t *testing.T) {
	sofaAddr := "127.0.0.1:8080"
	meshAddr := "127.0.0.1:2045"
	server := NewUpstreamServer(t, sofaAddr, ServeBoltV1)
	server.GoServe()
	defer server.Close()
	mesh_config := CreateSimpleMeshConfig(meshAddr, []string{sofaAddr}, protocol.SofaRpc, protocol.SofaRpc)
	go mosn.Start(mesh_config, "", "")
	time.Sleep(5 * time.Second) //wait mesh and server start
	//client
	client := &sofaClient{
		t:        t,
		ClientId: "testClient",
		Waits:    cmap.New(),
	}
	client.Connect(meshAddr)
	for i := 0; i < 20; i++ {
		client.SendRequest()
	}
	<-time.After(10 * time.Second)
	if !client.Waits.IsEmpty() {
		t.Errorf("exists request no response\n")
	}
}
