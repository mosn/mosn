package rpc

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/alipay/sofa-mosn/pkg/filter/network/proxy"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http2"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	"github.com/alipay/sofa-mosn/test/fuzzy"
	"github.com/alipay/sofa-mosn/test/util"
)

var (
	caseIndex    uint32 = 0
	caseDuration time.Duration
)

// this client needs verify response's status code
// do not care stream id
type RPCStatusClient struct {
	*util.RPCClient
	addr            string
	t               *testing.T
	mutex           sync.Mutex
	streamID        uint32
	unexpectedCount uint32
	successCount    uint32
	failureCount    uint32
	started         bool
}

func NewRPCClient(t *testing.T, id string, proto string, addr string) *RPCStatusClient {
	client := util.NewRPCClient(t, id, proto)
	return &RPCStatusClient{
		RPCClient: client,
		addr:      addr,
		t:         t,
		mutex:     sync.Mutex{},
	}
}

// over write
func (c *RPCStatusClient) SendRequest() {
	c.mutex.Lock()
	check := c.started
	c.mutex.Unlock()
	if !check {
		return
	}
	ID := atomic.AddUint32(&c.streamID, 1)
	streamID := protocol.StreamIDConv(ID)
	requestEncoder := c.Codec.NewStream(streamID, c)
	headers := util.BuildBoltV1Request(ID)
	requestEncoder.AppendHeaders(headers, true)
}

func (c *RPCStatusClient) OnReceiveHeaders(headers map[string]string, endStream bool) {
	status, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)]
	if !ok {
		c.t.Errorf("unexpected headers :%v\n", headers)
		c.unexpectedCount++
		return
	}
	code, err := strconv.Atoi(status)
	if err != nil {
		c.t.Errorf("unexpected status code: %s\n", status)
		c.unexpectedCount++
		return
	}
	if int16(code) == sofarpc.RESPONSE_STATUS_SUCCESS {
		c.successCount++
	} else {
		c.failureCount++
	}
}
func (c *RPCStatusClient) Connect() error {
	c.mutex.Lock()
	check := c.started
	c.mutex.Unlock()
	if check {
		return nil
	}
	if err := c.RPCClient.Connect(c.addr); err != nil {
		return err
	}
	c.mutex.Lock()
	c.started = true
	c.mutex.Unlock()
	return nil
}
func (c *RPCStatusClient) Close() {
	c.mutex.Lock()
	c.RPCClient.Close()
	c.started = false
	c.mutex.Unlock()
}
func (c *RPCStatusClient) RandomEvent(stop chan struct{}) {
	go func() {
		t := time.NewTicker(caseDuration / 5)
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				time.Sleep(util.RandomDuration(100*time.Millisecond, time.Second))
				switch rand.Intn(2) {
				case 0: //close
					c.Close()
					log.StartLogger.Infof("[FUZZY TEST] Close client #%s\n", c.ClientID)
				default: //
					log.StartLogger.Infof("[FUZZY TEST] Connect client #%s, error: %v\n", c.ClientID, c.Connect())
				}
			}
		}
	}()
}

type RPCServer struct {
	util.UpstreamServer
	t        *testing.T
	ID       string
	mutex    sync.Mutex
	started  bool
	finished bool
}

func NewRPCServer(t *testing.T, id string, addr string) *RPCServer {
	server := util.NewUpstreamServer(t, addr, util.ServeBoltV1)
	return &RPCServer{
		UpstreamServer: server,
		t:              t,
		ID:             id,
		mutex:          sync.Mutex{},
	}
}

//over write
func (s *RPCServer) Close(finished bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// once finished is set to true, it cannot be changed
	if !s.finished {
		s.finished = finished
	}
	if !s.started {
		return
	}
	log.StartLogger.Infof("[FUZZY TEST] server closed %s", s.ID)
	s.started = false
	s.UpstreamServer.Close()
}
func (s *RPCServer) GoServe() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		return
	}
	s.started = true
	s.UpstreamServer.GoServe()
}

func (s *RPCServer) ReStart() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		return
	}
	if s.finished {
		return
	}
	log.StartLogger.Infof("[FUZZY TEST] server restart #%s", s.ID)
	server := util.NewUpstreamServer(s.t, s.UpstreamServer.Addr(), util.ServeBoltV1)
	s.UpstreamServer = server
	s.started = true
	s.UpstreamServer.GoServe()
}

func CreateServers(t *testing.T, serverList []string, stop chan struct{}) []fuzzy.Server {
	var servers []fuzzy.Server
	for i, s := range serverList {
		id := fmt.Sprintf("server#%d", i)
		server := NewRPCServer(t, id, s)
		server.GoServe()
		go func(server *RPCServer) {
			<-stop
			log.StartLogger.Infof("[FUZZY TEST] finished fuzzy server %s", server.ID)
			server.Close(true)
		}(server)
		servers = append(servers, server)
	}
	return servers
}

//main
func TestMain(m *testing.M) {
	util.MeshLogPath = "./logs/rpc.log"
	util.MeshLogLevel = "INFO"
	log.InitDefaultLogger(util.MeshLogPath, log.INFO)
	casetime := flag.Int64("casetime", 1, "-casetime=1(min)")
	flag.Parse()
	caseDuration = time.Duration(*casetime) * time.Minute
	log.StartLogger.Infof("each case at least run %v", caseDuration)
	m.Run()
}
