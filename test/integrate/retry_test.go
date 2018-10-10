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
}

func NewRetryCase(t *testing.T, serverProto, meshProto types.Protocol) *RetryCase {
	app1 := "127.0.0.1:8080"
	app2 := "127.0.0.1:8081"
	var good, bad util.UpstreamServer
	switch serverProto {
	case protocol.HTTP1:
		good = util.NewHTTPServer(t, nil)
		bad = util.NewHTTPServer(t, &BadHTTPHandler{})
	case protocol.HTTP2:
		good = util.NewUpstreamHTTP2(t, app1, nil)
		bad = util.NewUpstreamHTTP2(t, app2, nil)
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
		tc,
		good,
		bad,
	}
}

func (c *RetryCase) Start(tls bool) {
	c.GoodServer.GoServe()
	c.BadServer.GoServe()
	app1 := c.GoodServer.Addr()
	app2 := c.BadServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(clientMeshAddr, serverMeshAddr, c.AppProtocol, c.MeshProtocol, []string{app1, app2}, tls)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Stop
		c.GoodServer.Close()
		c.BadServer.Close()
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
	testCases := []*RetryCase{
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP1),
		NewRetryCase(t, protocol.HTTP1, protocol.HTTP2),
		NewRetryCase(t, protocol.HTTP2, protocol.HTTP1),
		NewRetryCase(t, protocol.HTTP2, protocol.HTTP2),
		NewRetryCase(t, protocol.SofaRPC, protocol.HTTP1),
		NewRetryCase(t, protocol.SofaRPC, protocol.HTTP2),
		NewRetryCase(t, protocol.SofaRPC, protocol.SofaRPC),
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
