package proxy

import (
	"context"
	"strings"
	"testing"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/stream"
	_ "mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/streamfilter"
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
	callCreateFilterChain := false
	monkey.Patch(streamfilter.GetStreamFilterManager, func() streamfilter.StreamFilterManager {
		factory := streamfilter.NewMockStreamFilterFactory(ctrl)
		factory.EXPECT().CreateFilterChain(gomock.Any(), gomock.Any()).Do(func(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
			callCreateFilterChain = true
		}).AnyTimes()
		filterManager := streamfilter.NewMockStreamFilterManager(ctrl)
		filterManager.EXPECT().GetStreamFilterFactory(gomock.Any()).Return(factory).AnyTimes()
		return filterManager
	})
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
	monkey.Patch(stream.CreateServerStreamConnection, func(ctx context.Context, _ types.ProtocolName, _ api.Connection, proxy types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
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

func TestProxyFallbackNormal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	readCallback := mock.NewMockReadFilterCallbacks(ctrl)
	readCallback.EXPECT().Connection().AnyTimes().DoAndReturn(func() api.Connection {
		c := mock.NewMockConnection(ctrl)
		c.EXPECT().RawConn().AnyTimes().Return(nil)
		return c
	})

	var prot api.Protocol
	dispatch := 0
	monkey.Patch(stream.CreateServerStreamConnection, func(ctx context.Context, p api.Protocol, conn api.Connection,
		l types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
		prot = p
		ret := mock.NewMockServerStreamConnection(ctrl)
		ret.EXPECT().Dispatch(gomock.Any()).AnyTimes().DoAndReturn(func(buffer buffer.IoBuffer) {
			dispatch++
		})
		return ret
	})

	proxy := &proxy{
		config:           &v2.Proxy{FallbackForUnknownProtocol: true},
		readCallbacks:    readCallback,
		fallback:         false,
		serverStreamConn: nil,
		context:          context.TODO(),
	}

	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes([]byte("GET /"))), api.Stop)
	assert.False(t, proxy.fallback)
	assert.Equal(t, prot, protocol.HTTP1)
	assert.Equal(t, dispatch, 1)

	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes([]byte("GET /"))), api.Stop)
	assert.Equal(t, dispatch, 2)
}

func TestProxyFallbackDoFallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	readCallback := mock.NewMockReadFilterCallbacks(ctrl)
	readCallback.EXPECT().Connection().AnyTimes().DoAndReturn(func() api.Connection {
		c := mock.NewMockConnection(ctrl)
		c.EXPECT().RawConn().AnyTimes().Return(nil)
		return c
	})

	testInput := [][]byte{
		[]byte("helloWorldHelloWorld"), // normal case
		[]byte("h"),                    // small package
	}

	for _, in := range testInput {
		proxy := &proxy{
			config:           &v2.Proxy{FallbackForUnknownProtocol: true},
			readCallbacks:    readCallback,
			fallback:         false,
			serverStreamConn: nil,
		}

		assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes(in)), api.Continue)
		assert.True(t, proxy.fallback)

		// ensure panic if not return directly
		proxy.readCallbacks = nil
		proxy.serverStreamConn = nil
		assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes(in)), api.Continue)

		// once fallback, should never match protocol
		assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes([]byte("GET /"))), api.Continue)
	}
}
