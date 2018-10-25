package integrate

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
)

type RetryCase struct {
	*TestCase
	GoodServer util.UpstreamServer
	BadServer  util.UpstreamServer
	BadIsClose bool
}

func NewRetryCase(t *testing.T, serverProto, meshProto types.Protocol, isClose bool) *RetryCase {
	app1 := "127.0.0.1:8080"
	app2 := "127.0.0.1:8081"
	var good, bad util.UpstreamServer
	switch serverProto {
	case protocol.HTTP1:
		good = util.NewHTTPServer(t, nil)
		bad = util.NewHTTPServer(t, &BadHTTPHandler{})
	case protocol.HTTP2:
		good = util.NewUpstreamHTTP2(t, app1, nil)
		bad = util.NewUpstreamHTTP2(t, app2, &BadHTTPHandler{})
	case protocol.SofaRPC:
		good = util.NewRPCServer(t, app1, util.Bolt1)
		bad = util.RPCServer{
			Client:         util.NewRPCClient(t, "rpcClient", util.Bolt1),
			Name:           app2,
			UpstreamServer: util.NewUpstreamServer(t, app2, ServeBadBoltV1),
		}
	}
	tc := NewTestCase(t, serverProto, meshProto, util.NewRPCServer(t, "", util.Bolt1)) // Empty RPC server for get rpc client
	return &RetryCase{
		TestCase:   tc,
		GoodServer: good,
		BadServer:  bad,
		BadIsClose: isClose,
	}
}
func (c *RetryCase) StartProxy() {
	c.GoodServer.GoServe()
	c.BadServer.GoServe()
	app1 := c.GoodServer.Addr()
	app2 := c.BadServer.Addr()
	if c.BadIsClose {
		c.BadServer.Close()
	}
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	cfg := util.CreateProxyMesh(clientMeshAddr, []string{app1, app2}, c.AppProtocol)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.GoodServer.Close()
		if !c.BadIsClose {
			c.BadServer.Close()
		}
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start

}

func (c *RetryCase) Start(tls bool) {
	c.GoodServer.GoServe()
	c.BadServer.GoServe()
	app1 := c.GoodServer.Addr()
	app2 := c.BadServer.Addr()
	if c.BadIsClose {
		c.BadServer.Close()
	}
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(clientMeshAddr, serverMeshAddr, c.AppProtocol, c.MeshProtocol, []string{app1, app2}, tls)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.GoodServer.Close()
		if !c.BadIsClose {
			c.BadServer.Close()
		}
		mesh.Close()
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

// BadServer Handler
type BadHTTPHandler struct{}

func (h *BadHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	for k := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))
}

func ServeBadBoltV1(t *testing.T, conn net.Conn) {
	response := func(iobuf types.IoBuffer) ([]byte, bool) {
		cmd, _ := codec.BoltV1.GetDecoder().Decode(nil, iobuf)
		if cmd == nil {
			return nil, false
		}
		if req, ok := cmd.(*sofarpc.BoltRequestCommand); ok {
			resp := util.BuildBoltV1Response(req)
			resp.ResponseStatus = sofarpc.RESPONSE_STATUS_SERVER_EXCEPTION
			iobufresp, err := codec.BoltV1.GetEncoder().EncodeHeaders(nil, resp)
			if err != nil {
				t.Errorf("Build response error: %v\n", err)
				return nil, true
			}
			return iobufresp.Bytes(), true
		}
		return nil, true
	}
	util.ServeSofaRPC(t, conn, response)
}

func TestRetry(t *testing.T) {
	util.StartRetry = true
	defer func() {
		util.StartRetry = false
	}()
	testCases := []*RetryCase{
		// A server reponse not success
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP1, false),
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP2, false),
		NewRetryCase(t, protocol.HTTP2, protocol.HTTP1, false),
		NewRetryCase(t, protocol.HTTP2, protocol.HTTP2, false),
		NewRetryCase(t, protocol.SofaRPC, protocol.HTTP1, false),
		NewRetryCase(t, protocol.SofaRPC, protocol.HTTP2, false),
		NewRetryCase(t, protocol.SofaRPC, protocol.SofaRPC, false),
		// A server is shutdown
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP1, true),
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP2, true),
		// HTTP2 and SofaRPC will create connection to upstream before send request to upstream
		// If upstream is closed, it will failed directly, and we cannot do a retry before we send a request to upstream
		/*
			NewRetryCase(t, protocol.HTTP2, protocol.HTTP1, true),
			NewRetryCase(t, protocol.HTTP2, protocol.HTTP2, true),
			NewRetryCase(t, protocol.SofaRPC, protocol.HTTP1, true),
			NewRetryCase(t, protocol.SofaRPC, protocol.HTTP2, true),
			NewRetryCase(t, protocol.SofaRPC, protocol.SofaRPC, true),
		*/
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(false)
		// at least run twice
		go tc.RunCase(2, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		close(tc.Stop)
		time.Sleep(time.Second)
	}
}

func TestRetryProxy(t *testing.T) {
	util.StartRetry = true
	defer func() {
		util.StartRetry = false
	}()
	testCases := []*RetryCase{
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP1, false),
		NewRetryCase(t, protocol.HTTP2, protocol.HTTP2, false),
		NewRetryCase(t, protocol.SofaRPC, protocol.SofaRPC, false),
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP1, true),
		//NewRetryCase(t, protocol.HTTP2, protocol.HTTP2, true),
		//NewRetryCase(t, protocol.SofaRPC, protocol.SofaRPC, true),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartProxy()
		// at least run twice
		go tc.RunCase(2, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		close(tc.Stop)
		time.Sleep(time.Second)

	}
}
