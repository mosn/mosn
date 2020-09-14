package proxy

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

func TestNewProxy(t *testing.T) {
	// generate a basic context for new proxy
	genctx := func() context.Context {
		ctx := context.Background()
		ctx = mosnctx.WithValue(ctx, types.ContextKeyAccessLogs, []api.AccessLog{})
		ctx = mosnctx.WithValue(ctx, types.ContextKeyListenerName, "test_listener")
		return ctx
	}
	t.Run("config simple", func(t *testing.T) {
		// go mock
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// monkey patch
		// should use  -gcflags=-l makes monkey.patch valid. see https://github.com/bouk/monkey
		monkey.Patch(router.GetRoutersMangerInstance, func() types.RouterManager {
			rm := mock.NewMockRouterManager(ctrl)
			rm.EXPECT().GetRouterWrapperByName("test_router").DoAndReturn(func(_ string) types.RouterWrapper {
				rw := mock.NewMockRouterWrapper(ctrl)
				return rw
			})
			return rm
		})
		defer monkey.UnpatchAll()
		pv := NewProxy(genctx(), &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "Http1",
			UpstreamProtocol:   "Http1",
			RouterConfigName:   "test_router",
		})
		// verify
		p := pv.(*proxy)
		if p.routersWrapper == nil {
			t.Fatalf("should returns an router wrapper")
		}
	})

	t.Run("config with subprotocol", func(t *testing.T) {
		subs := "bolt,boltv2"
		pv := NewProxy(genctx(), &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "X",
			UpstreamProtocol:   "X",
			RouterConfigName:   "test_router",
			ExtendConfig: map[string]interface{}{
				"sub_protocol": subs,
			},
		})
		// verify
		p := pv.(*proxy)
		sub, ok := mosnctx.Get(p.context, types.ContextSubProtocol).(string)
		if !ok {
			t.Fatal("no sub protocol got")
		}
		if !strings.EqualFold(sub, subs) {
			t.Fatalf("got subprotocol %s, but expected %s", sub, subs)
		}
	})
	t.Run("config with proxy general", func(t *testing.T) {
		pv := NewProxy(genctx(), &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "Http1",
			UpstreamProtocol:   "Http1",
			RouterConfigName:   "test_router",
			ExtendConfig: map[string]interface{}{
				"http2_use_stream":      true,
				"max_request_body_size": 100,
			},
		})
		// verify
		p := pv.(*proxy)
		cfg, ok := mosnctx.Get(p.context, types.ContextKeyProxyGeneralConfig).(v2.ProxyGeneralExtendConfig)
		if !ok {
			t.Fatal("no proxy extend config")
		}
		if !(cfg.Http2UseStream &&
			cfg.MaxRequestBodySize == 100) {
			t.Fatalf("extend config is not expected, %+v", cfg)
		}
	})
}

// TestNewProxyRequest mocks a connection received a request and create a new proxy to handle it.
// NewProxy -> InitializeReadFilterCallbacks -> OnData -> NewStreamDetect(in stream.Dispatch)
func TestNewProxyRequest(t *testing.T) {
	// prepare mock
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()
	// generate a basic context for new proxy
	genctx := func() context.Context {
		ctx := context.Background()
		ctx = mosnctx.WithValue(ctx, types.ContextKeyAccessLogs, []api.AccessLog{})
		ctx = mosnctx.WithValue(ctx, types.ContextKeyListenerName, "test_listener")
		return ctx
	}
	pv := NewProxy(genctx(), &v2.Proxy{
		Name:               "test",
		DownstreamProtocol: "Http1",
		UpstreamProtocol:   "Http1",
		RouterConfigName:   "test_router",
	})
	//
	mockSpan := func() types.Span {
		sp := mock.NewMockSpan(ctrl)
		sp.EXPECT().TraceId().Return("1").AnyTimes()
		sp.EXPECT().SpanId().Return("1").AnyTimes()
		return sp
	}
	monkey.Patch(trace.IsEnabled, func() bool {
		return true
	})
	callCreateFilterChain := false
	factory := mock.NewMockStreamFilterChainFactory(ctrl)
	factory.EXPECT().CreateFilterChain(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ api.StreamFilterChainFactoryCallbacks) {
		callCreateFilterChain = true
	}).AnyTimes()
	value := new(atomic.Value)
	value.Store([]api.StreamFilterChainFactory{
		factory,
	})
	//
	monkey.Patch(stream.CreateServerStreamConnection, func(ctx context.Context, _ types.ProtocolName, _ api.Connection, proxy types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
		ctx = mosnctx.WithValue(ctx, types.ContextKeyStreamFilterChainFactories, value)
		sconn := mock.NewMockServerStreamConnection(ctrl)
		sconn.EXPECT().Dispatch(gomock.Any()).DoAndReturn(func(_ buffer.IoBuffer) {
			// sender is nil == oneway request
			proxy.NewStreamDetect(ctx, nil, mockSpan())
			return
		}).AnyTimes()
		sconn.EXPECT().Protocol().Return(types.ProtocolName("Http1")).AnyTimes()
		return sconn
	})
	cb := mock.NewMockReadFilterCallbacks(ctrl)
	cb.EXPECT().Connection().DoAndReturn(func() api.Connection {
		conn := mock.NewMockConnection(ctrl)
		conn.EXPECT().SetCollector(gomock.Any(), gomock.Any()).AnyTimes()
		conn.EXPECT().AddConnectionEventListener(gomock.Any()).AnyTimes()
		conn.EXPECT().LocalAddr().Return(nil).AnyTimes()  // mock, no use
		conn.EXPECT().RemoteAddr().Return(nil).AnyTimes() // mock, no use
		return conn
	}).AnyTimes()
	pv.InitializeReadFilterCallbacks(cb)
	// no auto protocol. stream conn will be setted before ondata
	if p := pv.(*proxy); p.serverStreamConn == nil {
		t.Fatal("no stream connection created")
	}
	pv.OnData(buffer.NewIoBuffer(0))
	// verify
	// stream filter chain is created
	if !callCreateFilterChain {
		t.Fatalf("no stream filter chain is created")
	}

}
