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
	"strconv"
	"strings"
	"testing"

	mhttp2 "mosn.io/mosn/pkg/module/http2"
	mhpack "mosn.io/mosn/pkg/module/http2/hpack"
	"mosn.io/mosn/pkg/network"
	phttp2 "mosn.io/mosn/pkg/protocol/http2"
	_ "mosn.io/mosn/pkg/proxy"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/buffer"
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
	variable.SetVariableValue(ctx, types.VarProxyIsDirectResponse, types.IsDirectResponse)

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
