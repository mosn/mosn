package integrate

import (
	"context"
	"testing"
	"time"

	"mosn.io/mosn/pkg/protocol/xprotocol/dubbothrift"

	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/protocol/xprotocol/tars"

	"mosn.io/mosn/pkg/module/http2"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/stream"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

func (c *TestCase) StartAuto(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(clientMeshAddr, serverMeshAddr, protocol.Auto, protocol.Auto, []string{appAddr}, tls)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestAuto(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartAuto(false)
		go tc.RunCase(5, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}
}

func TestAutoTLS(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	testCases := []*TestCase{
		NewTestCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, nil)),
		NewTestCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, nil)),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartAuto(true)
		go tc.RunCase(5, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}
}

func TestProtocolHttp2(t *testing.T) {
	var prot types.ProtocolName
	var magic string
	var err error

	magic = http2.ClientPreface
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(magic))
	if prot != protocol.HTTP2 {
		t.Errorf("[ERROR MESSAGE] type error magic : %v\n", magic)
	}

	len := len(http2.ClientPreface)
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(magic)[0:len-1])
	if err != stream.EAGAIN {
		t.Errorf("[ERROR MESSAGE] type error protocol :%v", err)
	}

	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte("helloworldhelloworldhelloworld"))
	if err != stream.FAILED {
		t.Errorf("[ERROR MESSAGE] type error protocol :%v", err)
	}
}

func TestProtocolHttp1(t *testing.T) {
	var prot types.ProtocolName
	var magic string
	var err error

	magic = "GET"
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(magic))
	if prot != protocol.HTTP1 {
		t.Errorf("[ERROR MESSAGE] type error magic : %v\n", magic)
	}

	magic = "POST"
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(magic))
	if prot != protocol.HTTP1 {
		t.Errorf("[ERROR MESSAGE] type error magic : %v\n", magic)
	}

	magic = "PATCH"
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(magic))
	if prot != protocol.HTTP1 {
		t.Errorf("[ERROR MESSAGE] type error magic : %v\n", magic)
	}

	magic = "POS"
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(magic))
	if err != stream.EAGAIN {
		t.Errorf("[ERROR MESSAGE] type error protocol :%v", err)
	}

	magic = "PPPPPPPPPPPPPPPPPPPPP"
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(magic))
	if err != stream.FAILED {
		t.Errorf("[ERROR MESSAGE] type error protocol :%v", err)
	}
}

func (c *XTestCase) StartXAuto(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := util.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(clientMeshAddr, serverMeshAddr, protocol.Auto, protocol.Auto, []string{appAddr}, tls)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestXAuto(t *testing.T) {

	appaddr := "127.0.0.1:20880"
	testCases := []*XTestCase{
		NewXTestCase(t, dubbo.ProtocolName, util.NewRPCServer(t, appaddr, dubbo.ProtocolName)),
		NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName)),
		NewXTestCase(t, dubbothrift.ProtocolName, util.NewRPCServer(t, appaddr, dubbothrift.ProtocolName)),
		NewXTestCase(t, tars.ProtocolName, util.NewRPCServer(t, appaddr, tars.ProtocolName)),
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.StartXAuto(false)
		go tc.RunCase(5, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}
}

func TestXProtocol(t *testing.T) {
	var prot types.ProtocolName
	var magic []byte
	var err error

	ctx := mosnctx.WithValue(context.Background(), types.ContextKeyStreamID, 1)

	magic = []byte{0xda, 0xbb, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	prot, err = stream.SelectStreamFactoryProtocol(ctx, "", magic)
	if prot != protocol.Xprotocol {
		t.Errorf("[ERROR MESSAGE] type error magic : %v\n", magic)
	}
	if string(dubbo.ProtocolName) != mosnctx.Get(ctx, types.ContextSubProtocol).(string) {
		t.Errorf("[ERROR MESSAGE] error sub protocol")
	}

	magic = []byte{0x1}
	prot, err = stream.SelectStreamFactoryProtocol(ctx, "", magic)
	if prot != protocol.Xprotocol {
		t.Errorf("[ERROR MESSAGE] type error magic : %v\n", magic)
	}
	if string(bolt.ProtocolName) != mosnctx.Get(ctx, types.ContextSubProtocol).(string) {
		t.Errorf("[ERROR MESSAGE] error sub protocol")
	}

	magic = []byte{0x00, 0x00, 0x00, 0x06, 0x10, 0x01}
	prot, err = stream.SelectStreamFactoryProtocol(ctx, "", magic)
	if prot != protocol.Xprotocol {
		t.Errorf("[ERROR MESSAGE] type error protocol :%v", err)
	}
	if string(tars.ProtocolName) != mosnctx.Get(ctx, types.ContextSubProtocol).(string) {
		t.Errorf("[ERROR MESSAGE] error sub protocol")
	}

	str := "PPPPPPPPPPPPPPPPPPPPP"
	prot, err = stream.SelectStreamFactoryProtocol(nil, "", []byte(str))
	if err != stream.FAILED {
		t.Errorf("[ERROR MESSAGE] type error protocol :%v", err)
	}
}
