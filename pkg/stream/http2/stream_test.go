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

package http2

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	monkey "github.com/cch123/supermonkey"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/mock"
	mhttp2 "mosn.io/mosn/pkg/module/http2"
	mhpack "mosn.io/mosn/pkg/module/http2/hpack"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	phttp2 "mosn.io/mosn/pkg/protocol/http2"
	_ "mosn.io/mosn/pkg/proxy"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"
)

func TestDirectResponse(t *testing.T) {

	testAddr := "127.0.0.1:11345"
	l, err := net.Listen("tcp", testAddr)
	if err != nil {
		t.Logf("listen error %v", err)
		return
	}
	defer l.Close()

	rawc, err := net.Dial("tcp", testAddr)
	if err != nil {
		t.Errorf("net.Dial error %v", err)
		return
	}

	connection := network.NewServerConnection(context.Background(), rawc, nil)

	// set direct response
	ctx := variable.NewVariableContext(context.Background())
	variable.SetString(ctx, types.VarProxyIsDirectResponse, types.IsDirectResponse)

	reqbodybuf := buffer.NewIoBufferString("1234567890")
	respbodybuf := buffer.NewIoBufferString("12345")

	mh := &mhttp2.MetaHeadersFrame{
		HeadersFrame: &mhttp2.HeadersFrame{
			FrameHeader: mhttp2.FrameHeader{
				Type:     mhttp2.FrameHeaders,
				Flags:    0,
				Length:   1,
				StreamID: 1,
			},
		},
		Fields: []mhpack.HeaderField(nil),
	}

	func(mh *mhttp2.MetaHeadersFrame, pairs ...string) {
		for len(pairs) > 0 {
			mh.Fields = append(mh.Fields, mhpack.HeaderField{
				Name:  pairs[0],
				Value: pairs[1],
			})
			pairs = pairs[2:]
		}
	}(mh, ":method", "GET", ":path", "/", ":scheme", "http", "Content-Length", strconv.Itoa(reqbodybuf.Len()))

	sc := newServerStreamConnection(ctx, connection, nil).(*serverStreamConnection)
	h2s, _, _, _, err := sc.sc.HandleFrame(ctx, mh)
	h2s.Response = &http.Response{
		Header: map[string][]string{},
	}
	if err != nil {
		t.Fatalf("handleFrame failed: %v", err)
	}

	s := &serverStream{
		h2s:    h2s,
		sc:     newServerStreamConnection(ctx, connection, nil).(*serverStreamConnection),
		stream: stream{ctx: ctx},
	}

	req := new(http.Request)
	req.Header = http.Header{}
	reqheader := phttp2.NewReqHeader(req)

	s.AppendHeaders(ctx, reqheader, false)
	s.AppendData(ctx, respbodybuf, true)

	encoder, _ := sc.sc.HeaderEncoder()
	got := fmt.Sprintf("%#v", encoder)

	want := strings.Trim(fmt.Sprintf("%#v", mhpack.HeaderField{
		Name:  "content-length",
		Value: strconv.Itoa(respbodybuf.Len()),
	}), "")

	if !strings.Contains(got, want) {
		t.Errorf("invalid content length got %s , want %s", got, want)
	}
}

func TestStreamConfigHandler(t *testing.T) {
	t.Run("test stream config", func(t *testing.T) {
		v := map[string]interface{}{
			"http2_use_stream": true,
		}
		rv := streamConfigHandler(v)
		cfg, ok := rv.(StreamConfig)
		if !ok {
			t.Fatalf("should returns StreamConfig but not")
		}
		if !cfg.Http2UseStream {
			t.Fatalf("invalid config: %v", cfg)
		}
	})
	t.Run("test invalid default", func(t *testing.T) {
		rv := streamConfigHandler(nil)
		cfg, ok := rv.(StreamConfig)
		if !ok {
			t.Fatalf("should returns StreamConfig but not")
		}
		if cfg.Http2UseStream {
			t.Fatalf("invalid config: %v", cfg)
		}
	})
}

func TestServerH2ReqUseStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testcases := []struct {
		dataFrame *mhttp2.DataFrame
		useStream bool
	}{
		{
			dataFrame: func() *mhttp2.DataFrame {
				dataFrame := &mhttp2.DataFrame{
					FrameHeader: mhttp2.FrameHeader{
						Type:     mhttp2.FrameData,
						Flags:    0,
						StreamID: 1,
					},
				}
				monkey.PatchInstanceMethod(reflect.TypeOf(dataFrame), "Data", func(f *mhttp2.DataFrame) []byte {
					return []byte("1234567890")
				})
				return dataFrame
			}(),
			useStream: true,
		},
		{
			dataFrame: func() *mhttp2.DataFrame {
				dataFrame := &mhttp2.DataFrame{
					FrameHeader: mhttp2.FrameHeader{
						Type:     mhttp2.FrameData,
						Flags:    mhttp2.FlagDataEndStream,
						StreamID: 3,
					},
				}
				monkey.PatchInstanceMethod(reflect.TypeOf(dataFrame), "Data", func(f *mhttp2.DataFrame) []byte {
					return []byte("1234567890")
				})
				return dataFrame
			}(),
			useStream: false,
		},
	}

	connection := mock.NewMockConnection(ctrl)
	connection.EXPECT().SetTransferEventListener(gomock.Any()).AnyTimes()
	connection.EXPECT().AddConnectionEventListener(gomock.Any()).AnyTimes()
	connection.EXPECT().RawConn().Return(nil).AnyTimes()
	connection.EXPECT().Write(gomock.Any()).AnyTimes().Return(nil)

	streamReceiver := mock.NewMockStreamReceiveListener(ctrl)

	var fp func(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, trailers api.HeaderMap)

	streamReceiver.EXPECT().OnReceive(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(testcases)).Do(
		func(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, trailers api.HeaderMap) {
			fp(ctx, headers, data, trailers)
		})

	http2Config := map[string]interface{}{
		"http2_use_stream": true,
	}
	proxyGeneralExtendConfig := make(map[api.ProtocolName]interface{})
	proxyGeneralExtendConfig[protocol.HTTP2] = streamConfigHandler(http2Config)
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableProxyGeneralConfig, proxyGeneralExtendConfig)

	serverCallbacks := mock.NewMockServerStreamConnectionEventListener(ctrl)

	serverCallbacks.EXPECT().NewStreamDetect(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(streamReceiver)

	sc := newServerStreamConnection(ctx, connection, serverCallbacks).(*serverStreamConnection)
	sc.cm.Next()

	monkey.PatchInstanceMethod(reflect.TypeOf(sc.sc), "HandleError", func(sc *mhttp2.MServerConn, ctx context.Context, f mhttp2.Frame, err error) {
		t.Fatalf("HandleError %v", err)
	})

	for _, testcase := range testcases {

		fp = func(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, trailers api.HeaderMap) {

			if useStream, err := variable.Get(ctx, types.VarHttp2RequestUseStream); err == nil {
				if h2UseStream, ok := useStream.(bool); ok {
					if h2UseStream && !testcase.useStream {
						t.Fatalf("unexpected use stream, expect: %v, actual: %v", testcase.useStream, useStream)
					}
				}
				return
			}
			if testcase.useStream {
				t.Fatalf("unexpected use stream, expect: %v, actual: %v", testcase.useStream, false)
			}
		}

		mh := &mhttp2.MetaHeadersFrame{
			HeadersFrame: &mhttp2.HeadersFrame{
				FrameHeader: mhttp2.FrameHeader{
					Type:     mhttp2.FrameHeaders,
					Flags:    0,
					Length:   1,
					StreamID: testcase.dataFrame.StreamID,
				},
			},
			Fields: []mhpack.HeaderField{
				{Name: ":method", Value: "GET"},
				{Name: ":path", Value: "/"},
				{Name: ":scheme", Value: "http"},
			},
		}

		ctx := sc.cm.Get()
		sc.handleFrame(ctx, mh, nil)
		sc.cm.Next()
		ctx = sc.cm.Get()
		sc.handleFrame(ctx, testcase.dataFrame, nil)
		sc.cm.Next()
	}
}

type mockProtocol struct {
	f func(ctx context.Context, model interface{}) (api.IoBuffer, error)
}

func (mp *mockProtocol) Name() types.ProtocolName {
	return "mockProtocolName"
}

func (mp *mockProtocol) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	return mp.f(ctx, model)
}

func (mp *mockProtocol) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	return nil, nil
}

func TestClientH2ReqUseStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testcases := []struct {
		useStream   bool
		retryEnable bool
	}{
		{
			useStream:   true,
			retryEnable: false,
		},
		{
			useStream:   false,
			retryEnable: true,
		},
	}

	http2Config := map[string]interface{}{
		"http2_use_stream": true,
	}
	proxyGeneralExtendConfig := make(map[api.ProtocolName]interface{})
	proxyGeneralExtendConfig[protocol.HTTP2] = streamConfigHandler(http2Config)
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableProxyGeneralConfig, proxyGeneralExtendConfig)

	connection := mock.NewMockConnection(ctrl)
	connection.EXPECT().AddConnectionEventListener(gomock.Any()).AnyTimes()
	connection.EXPECT().RawConn().Return(nil).AnyTimes()
	connection.EXPECT().Write(gomock.Any()).AnyTimes().Return(nil)
	connection.EXPECT().RemoteAddr().AnyTimes().Return(&net.TCPAddr{net.ParseIP("127.0.0.1"), 80, ""})

	clientCallbacks := mock.NewMockStreamConnectionEventListener(ctrl)

	responseReceiveListener := mock.NewMockStreamReceiveListener(ctrl)

	sc := newClientStreamConnection(ctx, connection, clientCallbacks).(*clientStreamConnection)

	sc.protocol = &mockProtocol{}

	req, _ := http.NewRequest("GET", "http://127.0.0.1:80/", nil)

	for _, testcase := range testcases {
		protocol := sc.protocol.(*mockProtocol)
		protocol.f = func(ctx context.Context, model interface{}) (api.IoBuffer, error) {
			if h2s, ok := model.(*mhttp2.MClientStream); !ok {
				t.Fatalf("invalid h2s type")
			} else {
				monkey.PatchInstanceMethod(reflect.TypeOf(h2s), "GetID", func(cc *mhttp2.MClientStream) uint32 {
					return 1
				})
				if h2s.UseStream != testcase.useStream {
					t.Fatalf("unexpected use stream, expect: %v, actual: %v", testcase.useStream, h2s.UseStream)
				}
			}
			return nil, nil
		}
		ctx := variable.NewVariableContext(ctx)
		variable.Set(ctx, types.VarHttp2RequestUseStream, testcase.useStream)

		clientStream := sc.NewStream(ctx, responseReceiveListener)
		err := clientStream.AppendHeaders(ctx, &phttp2.ReqHeader{
			HeaderMap: &phttp2.HeaderMap{
				H: req.Header,
			},
			Req: req,
		}, false)
		assert.Nil(t, err)
		clientStream.AppendTrailers(ctx, nil)
		var retryDisabled bool
		if disable, err := variable.Get(ctx, types.VarProxyDisableRetry); err == nil {
			if retryDisable, ok := disable.(bool); ok && retryDisable {
				retryDisabled = true
			}
		}
		if retryDisabled != !testcase.retryEnable {
			t.Fatalf("unexpected retry disable expected: %v, actual: %v", !testcase.retryEnable, retryDisabled)
		}
	}
}

func TestClientH2RespUseStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testcases := []struct {
		dataFrame *mhttp2.DataFrame
		useStream bool
	}{
		{
			dataFrame: func() *mhttp2.DataFrame {
				dataFrame := &mhttp2.DataFrame{
					FrameHeader: mhttp2.FrameHeader{
						Type:  mhttp2.FrameData,
						Flags: 0,
					},
				}
				monkey.PatchInstanceMethod(reflect.TypeOf(dataFrame), "Data", func(f *mhttp2.DataFrame) []byte {
					return []byte("1234567890")
				})
				return dataFrame
			}(),
			useStream: true,
		},
		{
			dataFrame: func() *mhttp2.DataFrame {
				dataFrame := &mhttp2.DataFrame{
					FrameHeader: mhttp2.FrameHeader{
						Type:  mhttp2.FrameData,
						Flags: mhttp2.FlagDataEndStream,
					},
				}
				monkey.PatchInstanceMethod(reflect.TypeOf(dataFrame), "Data", func(f *mhttp2.DataFrame) []byte {
					return []byte("1234567890")
				})
				return dataFrame
			}(),
			useStream: false,
		},
	}

	connection := mock.NewMockConnection(ctrl)
	connection.EXPECT().SetTransferEventListener(gomock.Any()).AnyTimes()
	connection.EXPECT().AddConnectionEventListener(gomock.Any()).AnyTimes()
	connection.EXPECT().RawConn().Return(nil).AnyTimes()
	connection.EXPECT().Write(gomock.Any()).AnyTimes().Return(nil)
	connection.EXPECT().Close(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	streamReceiver := mock.NewMockStreamReceiveListener(ctrl)

	var fp func(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, trailers api.HeaderMap)

	streamReceiver.EXPECT().OnReceive(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(testcases)).Do(
		func(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, trailers api.HeaderMap) {
			fp(ctx, headers, data, trailers)
		})

	http2Config := map[string]interface{}{
		"http2_use_stream": true,
	}
	proxyGeneralExtendConfig := make(map[api.ProtocolName]interface{})
	proxyGeneralExtendConfig[protocol.HTTP2] = streamConfigHandler(http2Config)
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableProxyGeneralConfig, proxyGeneralExtendConfig)

	clientCallbacks := mock.NewMockStreamConnectionEventListener(ctrl)

	sc := newClientStreamConnection(ctx, connection, clientCallbacks).(*clientStreamConnection)
	sc.cm.Next()

	//monkey.PatchInstanceMethod(reflect.TypeOf(sc.mClientConn), "HandleError", func(sc *mhttp2.MClientConn, ctx context.Context, f mhttp2.Frame, err error) {
	//	t.Fatalf("HandleError %v", err)
	//})
	req, _ := http.NewRequest("GET", "http://127.0.0.1:80/", nil)

	for _, testcase := range testcases {
		ctx = sc.cm.Get()
		mClientStream, _ := sc.mClientConn.WriteHeaders(ctx, req, "", false)
		testcase.dataFrame.StreamID = mClientStream.ID
		sClientStream := sc.NewStream(ctx, streamReceiver).(*clientStream)
		sClientStream.id = mClientStream.ID
		sc.streams[sClientStream.id] = sClientStream

		fp = func(ctx context.Context, headers api.HeaderMap, data buffer.IoBuffer, trailers api.HeaderMap) {
			if useStream, err := variable.Get(ctx, types.VarHttp2ResponseUseStream); err == nil {
				if h2UseStream, ok := useStream.(bool); ok {
					if h2UseStream && !testcase.useStream {
						t.Fatalf("unexpected use stream, expect: %v, actual: %v", testcase.useStream, useStream)
					}
					return
				}
			}
			if testcase.useStream {
				t.Fatalf("unexpected use stream, expect: %v, actual: %v", testcase.useStream, false)
			}
		}

		mh := &mhttp2.MetaHeadersFrame{
			HeadersFrame: &mhttp2.HeadersFrame{
				FrameHeader: mhttp2.FrameHeader{
					Type:     mhttp2.FrameHeaders,
					Flags:    0,
					Length:   1,
					StreamID: testcase.dataFrame.StreamID,
				},
			},
			Fields: []mhpack.HeaderField{
				{Name: ":status", Value: "200"},
			},
		}
		sc.cm.Next()
		ctx = sc.cm.Get()
		sc.handleFrame(ctx, mh, nil)
		sc.cm.Next()
		ctx = sc.cm.Get()
		sc.handleFrame(ctx, testcase.dataFrame, nil)
		sc.cm.Next()
	}
}

func TestServerH2RespUseStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testcases := []struct {
		useStream bool
	}{
		{
			useStream: true,
		},
		{
			useStream: false,
		},
	}

	http2Config := map[string]interface{}{
		"http2_use_stream": true,
	}
	proxyGeneralExtendConfig := make(map[api.ProtocolName]interface{})
	proxyGeneralExtendConfig[protocol.HTTP2] = streamConfigHandler(http2Config)
	ctx := variable.NewVariableContext(context.Background())
	_ = variable.Set(ctx, types.VariableProxyGeneralConfig, proxyGeneralExtendConfig)

	connection := mock.NewMockConnection(ctrl)
	connection.EXPECT().SetTransferEventListener(gomock.Any()).AnyTimes()
	connection.EXPECT().AddConnectionEventListener(gomock.Any()).AnyTimes()
	connection.EXPECT().RawConn().Return(nil).AnyTimes()
	connection.EXPECT().Write(gomock.Any()).AnyTimes().Return(nil)
	connection.EXPECT().RemoteAddr().AnyTimes().Return(&net.TCPAddr{net.ParseIP("127.0.0.1"), 80, ""})

	serverCallbacks := mock.NewMockServerStreamConnectionEventListener(ctrl)

	serverCallbacks.EXPECT().NewStreamDetect(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	//responseReceiveListener := mock.NewMockStreamReceiveListener(ctrl)

	sc := newServerStreamConnection(ctx, connection, serverCallbacks).(*serverStreamConnection)

	sc.protocol = &mockProtocol{}

	rsp := &http.Response{
		Header: http.Header{},
	}

	for _, testcase := range testcases {
		protocol := sc.protocol.(*mockProtocol)
		protocol.f = func(ctx context.Context, model interface{}) (api.IoBuffer, error) {
			if h2s, ok := model.(*mhttp2.MStream); !ok {
				t.Fatalf("invalid h2s type")
			} else {
				if h2s.UseStream != testcase.useStream {
					t.Fatalf("unexpected use stream, expect: %v, actual: %v", testcase.useStream, h2s.UseStream)
				}
			}
			return nil, nil
		}
		// mock Next, stream context inherit by connection context
		sctx := variable.NewVariableContext(ctx)
		variable.Set(sctx, types.VarHttp2ResponseUseStream, testcase.useStream)

		h2s := &mhttp2.MStream{}
		monkey.PatchInstanceMethod(reflect.TypeOf(h2s), "ID", func(cc *mhttp2.MStream) uint32 {
			return 1
		})
		serverStream, _ := sc.onNewStreamDetect(sctx, h2s, false)
		err := serverStream.AppendHeaders(sctx, &phttp2.RspHeader{
			HeaderMap: &phttp2.HeaderMap{
				H: rsp.Header,
			},
			Rsp: rsp,
		}, false)
		assert.Nil(t, err)
		serverStream.AppendTrailers(sctx, nil)
	}
}
