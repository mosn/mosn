package tests

import (
	"net"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/orcaman/concurrent-map"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/mosn"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

//Downstream
//types.StreamReceiver
type sofaClient struct {
	t               *testing.T
	ClientId        string
	CurrentStreamId uint32
	Codec           stream.CodecClient
	RequestCount    uint32
	ResponseCount   uint32
	Queue           cmap.ConcurrentMap
}

func (c *sofaClient) OnEvent(event types.ConnectionEvent) {
	if event.IsClose() {
		c.t.Logf("Closed Client %v", c.ClientId)
	}
}

func (c *sofaClient) Connect(addr string, stopChan chan bool) error {
	remoteAddr, _ := net.ResolveTCPAddr("tcp", addr)
	cc := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	if err := cc.Connect(true); err != nil {
		c.t.Errorf("client[%s] connect to server error: %v\n", c.ClientId, err)
		return err
	}
	c.Codec = stream.NewCodecClient(nil, protocol.SofaRpc, cc, nil)
	if c.Codec == nil {
		c.t.Errorf("client[%s] create codec failed", c.ClientId)
	}
	c.Codec.AddConnectionCallbacks(c)
	c.t.Logf("client[%s] connected to server: %s\n", c.ClientId, addr)
	return nil
}

func (c *sofaClient) SendRequest() {
	id := atomic.AddUint32(&c.CurrentStreamId, 1)
	streamId := sofarpc.StreamIDConvert(id)
	requestEncoder := c.Codec.NewStream(streamId, c)
	headers := buildRequestMsg(id)
	//Send Request
	requestEncoder.AppendHeaders(headers, true)
	//Record
	ch := make(chan struct{})
	c.Queue.Set(streamId, ch)
	atomic.AddUint32(&c.RequestCount, 1)
	//wait Response
	go func() {
		select {
		case <-ch:
			atomic.AddUint32(&c.ResponseCount, 1)
		case <-time.After(10 * time.Second):
			c.t.Errorf("client[%s] wait response timeout at stream[%s]\n", c.ClientId, streamId)
		}
	}()
}

func (c *sofaClient) OnReceiveHeaders(headers map[string]string, endStream bool) {
	streamId, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)]
	if ok {
		if resp, ok := c.Queue.Get(streamId); ok {
			if ch, ok := resp.(chan struct{}); ok {
				c.t.Logf("Get Stream Response: %s ,headers: %v\n", streamId, headers)
				c.Queue.Remove(streamId)
				close(ch)
			}
		} else {
			c.t.Logf("client[%s] response unknown stream, Id %s\n", c.ClientId, streamId)
		}
	}
	if status, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)]; ok {
		if sofarpc.ConvertPropertyValue(status, reflect.Int16).(int16) == 0 {
			//c.t.Logf("client[%s] Request %s Get Success Resposne\n", c.ClientId, streamId)
		} else {
			c.t.Errorf("client[%s] Request %s Get Status: %s\n", c.ClientId, streamId, status)
		}
	}
}

func (c *sofaClient) OnReceiveData(data types.IoBuffer, endStream bool) {
}
func (c *sofaClient) OnReceiveTrailers(trailers map[string]string) {
}
func (c *sofaClient) OnDecodeError(err error, headers map[string]string) {
}

func ServeBoltV1(t *testing.T, conn net.Conn) {
	iobuf := buffer.NewIoBuffer(102400)
	for {
		now := time.Now()
		conn.SetReadDeadline(now.Add(30 * time.Second))
		buf := make([]byte, 10*1024)
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				t.Logf("Connect read error: %v\n", err)
				continue
			}
			return
		}
		if bytesRead > 0 {
			iobuf.Write(buf[:bytesRead])
			for iobuf.Len() > 1 {
				_, cmd := codec.BoltV1.GetDecoder().Decode(nil, iobuf)
				if cmd == nil {
					break
				}
				if req, ok := cmd.(*sofarpc.BoltRequestCommand); ok {
					resp := buildRespMag(req)
					err, iobufresp := codec.BoltV1.GetEncoder().EncodeHeaders(nil, resp)
					if err != nil {
						t.Errorf("Build response error: %v\n", err)
					} else {
						t.Logf("server %s write to remote: %d\n", conn.LocalAddr().String(), resp.GetReqId)
						respdata := iobufresp.Bytes()
						conn.Write(respdata)
					}
				}
			}
		}
	}

}

func TestServerClose(t *testing.T) {
	serverAddrs := []string{
		"127.0.0.1:8080",
		"127.0.0.1:8081",
	}
	//Start Upstream
	servers := []*UpstreamServer{}
	for _, addr := range serverAddrs {
		s := NewUpstreamServer(t, addr, ServeBoltV1)
		s.GoServe()
		servers = append(servers, s)
	}
	//Start Mesh
	MeshAddr := "127.0.0.1:2045"
	mesh_config := CreateSofaRpcConfig(MeshAddr, serverAddrs)
	go mosn.Start(mesh_config, "", "")
	//Wait Server start
	time.Sleep(5 * time.Second)
	client := &sofaClient{
		t:        t,
		ClientId: "TestClient",
		Queue:    cmap.New(),
	}
	stopChan := make(chan bool)
	if err := client.Connect(MeshAddr, stopChan); err != nil {
		return
	}
	//持续发送请求
	go func() {
		for i := 0; i < 10; i++ {
			client.SendRequest()
			time.Sleep(time.Second)
		}
	}()
	//关闭一个Server
	go func() {
		<-time.After(3 * time.Second)
		servers[0].Close()
	}()
	//运行一段时间，等待响应
	<-time.After(20 * time.Second)
	t.Logf("client[%s] send %d requests, get %d response\n", client.ClientId, client.RequestCount, client.ResponseCount)
}

func TestMain(m *testing.M) {
	log.InitDefaultLogger("", log.DEBUG)
	m.Run()
}
