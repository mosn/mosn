/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package proxy

import (
	"context"
	"os"
	"testing"

	monkey "github.com/cch123/supermonkey"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mock"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/stream"
	"mosn.io/mosn/pkg/stream/http"
	"mosn.io/mosn/pkg/stream/http2"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func TestMain(m *testing.M) {
	// mock register variable
	variable.Register(variable.NewVariable(types.VarProtocolConfig, nil, nil, variable.DefaultSetter, 0))
	os.Exit(m.Run())
}

func TestNewProxy(t *testing.T) {
	// generate a basic context for new proxy
	genctx := func() context.Context {
		ctx := variable.NewVariableContext(context.Background())
		_ = variable.Set(ctx, types.VariableAccessLogs, []api.AccessLog{})
		_ = variable.Set(ctx, types.VariableListenerName, "test_listener")
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
		ctx := genctx()
		variable.Set(ctx, types.VarProtocolConfig, []api.ProtocolName{api.ProtocolName("Http1")})
		pv := NewProxy(ctx, &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "Http1",
			RouterConfigName:   "test_router",
		})
		// verify
		p := pv.(*proxy)
		if p.routersWrapper == nil {
			t.Fatalf("should returns an router wrapper")
		}
	})

	t.Run("config with subprotocol", func(t *testing.T) {
		subs := []api.ProtocolName{api.ProtocolName("bolt"), api.ProtocolName("boltv2")}
		ctx := genctx()
		variable.Set(ctx, types.VarProtocolConfig, subs)
		pv := NewProxy(ctx, &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "X",
			RouterConfigName:   "test_router",
		})
		// verify
		p := pv.(*proxy)
		if len(p.protocols) != 2 || p.protocols[0] != api.ProtocolName("bolt") || p.protocols[1] != api.ProtocolName("boltv2") {
			t.Fatalf("got subprotocol %v, but expected %v", p.protocols, subs)
		}
	})
	t.Run("config with proxy general HTTP1", func(t *testing.T) {
		ctx := genctx()
		variable.Set(ctx, types.VarProtocolConfig, []api.ProtocolName{api.ProtocolName("Http1")})
		cfg := &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "Http1",
			UpstreamProtocol:   "Http1",
			RouterConfigName:   "test_router",
			ExtendConfig: map[string]interface{}{
				"http2_use_stream":      true,
				"max_request_body_size": 100,
			},
		}
		// mock create proxy factory
		extConfig := make(map[api.ProtocolName]interface{})
		extConfig[api.ProtocolName(cfg.DownstreamProtocol)] = protocol.HandleConfig(api.ProtocolName(cfg.DownstreamProtocol), cfg.ExtendConfig)
		_ = variable.Set(ctx, types.VariableProxyGeneralConfig, extConfig)
		pv := NewProxy(ctx, cfg)
		// verify
		p := pv.(*proxy)
		var v http.StreamConfig
		if pgc, err := variable.Get(p.context, types.VariableProxyGeneralConfig); err == nil {
			if extendConfig, ok := pgc.(map[api.ProtocolName]interface{}); ok {
				if http1Config, ok := extendConfig[protocol.HTTP1]; ok {
					if cfg, ok := http1Config.(http.StreamConfig); ok {
						v = cfg
					}
				}
			}
		}
		check := func(t *testing.T, v interface{}) {
			cfg, ok := v.(http.StreamConfig)
			assert.True(t, ok)
			assert.Equal(t, 100, cfg.MaxRequestBodySize)
		}
		check(t, v)
	})
	t.Run("config with proxy general http1 and bolt", func(t *testing.T) {
		subs := []api.ProtocolName{api.ProtocolName("Http1"), api.ProtocolName("Http2")}
		ctx := genctx()
		variable.Set(ctx, types.VarProtocolConfig, subs)
		cfg := &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "Http1,Http2",
			ExtendConfig: map[string]interface{}{
				"Http1": map[string]interface{}{
					"http2_use_stream":      true,
					"max_request_body_size": 100,
				},
				"Http2": map[string]interface{}{
					"http2_use_stream": true,
				},
			},
		}
		// mock create proxy factory
		extConfig := make(map[api.ProtocolName]interface{})
		for proto, _ := range cfg.ExtendConfig {
			extConfig[api.ProtocolName(proto)] = protocol.HandleConfig(api.ProtocolName(proto), cfg.ExtendConfig[proto])
		}
		_ = variable.Set(ctx, types.VariableProxyGeneralConfig, extConfig)
		pv := NewProxy(ctx, cfg)
		// verify
		p := pv.(*proxy)
		var value1 http.StreamConfig
		var value2 http2.StreamConfig
		if pgc, err := variable.Get(p.context, types.VariableProxyGeneralConfig); err == nil {
			if extendConfig, ok := pgc.(map[api.ProtocolName]interface{}); ok {
				if http1Config, ok := extendConfig[protocol.HTTP1]; ok {
					if cfg, ok := http1Config.(http.StreamConfig); ok {
						value1 = cfg
					}
				}
			}
			if extendConfig, ok := pgc.(map[api.ProtocolName]interface{}); ok {
				if http2Config, ok := extendConfig[protocol.HTTP2]; ok {
					if cfg, ok := http2Config.(http2.StreamConfig); ok {
						value2 = cfg
					}
				}
			}
		}
		check := func(t *testing.T, value1 interface{}, value2 interface{}) {
			cfg1, ok := value1.(http.StreamConfig)
			assert.True(t, ok)
			assert.Equal(t, 100, cfg1.MaxRequestBodySize)
			cfg2, ok := value2.(http2.StreamConfig)
			assert.True(t, ok)
			assert.Equal(t, true, cfg2.Http2UseStream)
		}
		check(t, value1, value2)
	})
	t.Run("config with proxy general Auto(single protocol)", func(t *testing.T) {
		ctx := genctx()
		variable.Set(ctx, types.VarProtocolConfig, []api.ProtocolName{api.ProtocolName("Auto")})
		cfg := &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "Auto",
			UpstreamProtocol:   "Auto",
			RouterConfigName:   "test_router",
			ExtendConfig: map[string]interface{}{
				"Http1": map[string]interface{}{
					"http2_use_stream":      true,
					"max_request_body_size": 100,
				},
			},
		}
		// mock create proxy factory
		extConfig := make(map[api.ProtocolName]interface{})
		for proto, _ := range cfg.ExtendConfig {
			extConfig[api.ProtocolName(proto)] = protocol.HandleConfig(api.ProtocolName(proto), cfg.ExtendConfig[proto])
		}
		_ = variable.Set(ctx, types.VariableProxyGeneralConfig, extConfig)
		pv := NewProxy(ctx, cfg)
		// verify
		p := pv.(*proxy)
		var v http.StreamConfig
		if pgc, err := variable.Get(p.context, types.VariableProxyGeneralConfig); err == nil {
			if extendConfig, ok := pgc.(map[api.ProtocolName]interface{}); ok {
				if http1Config, ok := extendConfig[protocol.HTTP1]; ok {
					if cfg, ok := http1Config.(http.StreamConfig); ok {
						v = cfg
					}
				}
			}
		}
		check := func(t *testing.T, v interface{}) {
			cfg, ok := v.(http.StreamConfig)
			assert.True(t, ok)
			assert.Equal(t, 100, cfg.MaxRequestBodySize)
		}
		check(t, v)
	})
	t.Run("config with proxy general Auto(multi protocol)", func(t *testing.T) {
		ctx := genctx()
		variable.Set(ctx, types.VarProtocolConfig, []api.ProtocolName{api.ProtocolName("Http1")})
		cfg := &v2.Proxy{
			Name:               "test",
			DownstreamProtocol: "Auto",
			UpstreamProtocol:   "Auto",
			RouterConfigName:   "test_router",
			ExtendConfig: map[string]interface{}{
				"Http1": map[string]interface{}{
					"http2_use_stream":      true,
					"max_request_body_size": 100,
				},
				"Http2": map[string]interface{}{
					"http2_use_stream": true,
				},
			},
		}
		// mock create proxy factory
		extConfig := make(map[api.ProtocolName]interface{})
		for proto, _ := range cfg.ExtendConfig {
			extConfig[api.ProtocolName(proto)] = protocol.HandleConfig(api.ProtocolName(proto), cfg.ExtendConfig[proto])
		}
		_ = variable.Set(ctx, types.VariableProxyGeneralConfig, extConfig)
		pv := NewProxy(ctx, cfg)
		// verify
		p := pv.(*proxy)
		var value1 http.StreamConfig
		var value2 http2.StreamConfig
		if pgc, err := variable.Get(p.context, types.VariableProxyGeneralConfig); err == nil {
			if extendConfig, ok := pgc.(map[api.ProtocolName]interface{}); ok {
				if http1Config, ok := extendConfig[protocol.HTTP1]; ok {
					if cfg, ok := http1Config.(http.StreamConfig); ok {
						value1 = cfg
					}
				}
			}
			if extendConfig, ok := pgc.(map[api.ProtocolName]interface{}); ok {
				if http2Config, ok := extendConfig[protocol.HTTP2]; ok {
					if cfg, ok := http2Config.(http2.StreamConfig); ok {
						value2 = cfg
					}
				}
			}
		}
		check := func(t *testing.T, value1 interface{}, value2 interface{}) {
			cfg1, ok := value1.(http.StreamConfig)
			assert.True(t, ok)
			assert.Equal(t, 100, cfg1.MaxRequestBodySize)
			cfg2, ok := value2.(http2.StreamConfig)
			assert.True(t, ok)
			assert.Equal(t, true, cfg2.Http2UseStream)
		}
		check(t, value1, value2)
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
		ctx := variable.NewVariableContext(context.Background())
		_ = variable.Set(ctx, types.VariableAccessLogs, []api.AccessLog{})
		_ = variable.Set(ctx, types.VariableListenerName, "test_listener")
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
	ctx := genctx()
	variable.Set(ctx, types.VarProtocolConfig, []api.ProtocolName{api.ProtocolName("Http1")})
	pv := NewProxy(ctx, &v2.Proxy{
		Name:               "test",
		DownstreamProtocol: "Http1",
		UpstreamProtocol:   "Http1",
		RouterConfigName:   "test_router",
	})
	//
	mockSpan := func() api.Span {
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
	defer monkey.UnpatchAll()

	readCallback := mock.NewMockReadFilterCallbacks(ctrl)
	readCallback.EXPECT().Connection().AnyTimes().DoAndReturn(func() api.Connection {
		c := mock.NewMockConnection(ctrl)
		c.EXPECT().RawConn().AnyTimes().Return(nil)
		return c
	})

	var prot types.ProtocolName
	dispatch := 0
	monkey.Patch(stream.CreateServerStreamConnection, func(ctx context.Context, p types.ProtocolName, conn api.Connection,
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

	proxy := &proxy{
		config:           &v2.Proxy{FallbackForUnknownProtocol: true},
		readCallbacks:    readCallback,
		fallback:         false,
		serverStreamConn: nil,
	}

	data := []byte("helloWorldHelloWorld")
	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes(data)), api.Continue)
	assert.True(t, proxy.fallback)

	// ensure panic if not return directly
	proxy.readCallbacks = nil
	proxy.serverStreamConn = nil
	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes(data)), api.Continue)

	// once fallback, should never match protocol
	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes([]byte("GET /"))), api.Continue)
}

func TestProxyFallbackSmallPackage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	buff := buffer.NewIoBuffer(0)
	readCallback := mock.NewMockReadFilterCallbacks(ctrl)
	readCallback.EXPECT().Connection().AnyTimes().DoAndReturn(func() api.Connection {
		c := mock.NewMockConnection(ctrl)
		c.EXPECT().RawConn().AnyTimes().Return(nil)
		c.EXPECT().GetReadBuffer().AnyTimes().Return(buff)
		return c
	})
	continueReading := false
	readCallback.EXPECT().ContinueReading().AnyTimes().DoAndReturn(func() {
		continueReading = true
	})

	proxy := &proxy{
		config:           &v2.Proxy{FallbackForUnknownProtocol: true},
		readCallbacks:    readCallback,
		fallback:         false,
		serverStreamConn: nil,
	}
	proxy.downstreamListener = &downstreamCallbacks{
		proxy: proxy,
	}

	// wait for more data or timeout
	_, _ = buff.Write([]byte("b"))
	assert.Equal(t, proxy.OnData(buff), api.Stop)
	assert.False(t, proxy.fallback)

	// still wait for more data or timeout
	_, _ = buff.Write([]byte("b"))
	assert.Equal(t, proxy.OnData(buff), api.Stop)
	assert.False(t, proxy.fallback)

	// timeout
	proxy.downstreamListener.OnEvent(api.OnReadTimeout)
	assert.True(t, continueReading)
	assert.True(t, proxy.fallback)

	// ensure panic if not return directly
	proxy.readCallbacks = nil
	proxy.serverStreamConn = nil
	assert.Equal(t, proxy.OnData(buff), api.Continue)

	// once fallback, should never match protocol
	assert.Equal(t, proxy.OnData(buff), api.Continue)
}

func TestProxyFallbackSmallHTTPPackage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	readCallback := mock.NewMockReadFilterCallbacks(ctrl)
	readCallback.EXPECT().Connection().AnyTimes().DoAndReturn(func() api.Connection {
		c := mock.NewMockConnection(ctrl)
		c.EXPECT().RawConn().AnyTimes().Return(nil)
		return c
	})

	var prot types.ProtocolName
	monkey.Patch(stream.CreateServerStreamConnection, func(ctx context.Context, p types.ProtocolName, conn api.Connection,
		l types.ServerStreamConnectionEventListener) types.ServerStreamConnection {
		prot = p
		ret := mock.NewMockServerStreamConnection(ctrl)
		ret.EXPECT().Dispatch(gomock.Any()).AnyTimes().Return()
		return ret
	})

	proxy := &proxy{
		config:           &v2.Proxy{FallbackForUnknownProtocol: true},
		readCallbacks:    readCallback,
		fallback:         false,
		serverStreamConn: nil,
	}
	proxy.downstreamListener = &downstreamCallbacks{
		proxy: proxy,
	}

	// wait for more data or timeout
	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes([]byte("G"))), api.Stop)
	assert.False(t, proxy.fallback)

	// still wait for more data or timeout
	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes([]byte("GE"))), api.Stop)
	assert.False(t, proxy.fallback)

	// should auto match http1
	assert.Equal(t, proxy.OnData(buffer.NewIoBufferBytes([]byte("GET /"))), api.Stop)
	assert.False(t, proxy.fallback)
	assert.Equal(t, prot, protocol.HTTP1)

	// timeout
	proxy.downstreamListener.OnEvent(api.OnReadTimeout)
	assert.False(t, proxy.fallback)
}

func TestProxyFallbackTimeoutWithoutData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	buff := buffer.NewIoBuffer(0)
	readCallback := mock.NewMockReadFilterCallbacks(ctrl)
	readCallback.EXPECT().Connection().AnyTimes().DoAndReturn(func() api.Connection {
		c := mock.NewMockConnection(ctrl)
		c.EXPECT().RawConn().AnyTimes().Return(nil)
		c.EXPECT().GetReadBuffer().AnyTimes().Return(buff)
		return c
	})
	continueReading := false
	readCallback.EXPECT().ContinueReading().AnyTimes().DoAndReturn(func() {
		continueReading = true
	})

	proxy := &proxy{
		config:           &v2.Proxy{FallbackForUnknownProtocol: true},
		readCallbacks:    readCallback,
		fallback:         false,
		serverStreamConn: nil,
	}
	proxy.downstreamListener = &downstreamCallbacks{
		proxy: proxy,
	}

	// do not have any data and then timeout, should not fallback
	proxy.downstreamListener.OnEvent(api.OnReadTimeout)
	assert.False(t, proxy.fallback)
	assert.False(t, continueReading)
}
