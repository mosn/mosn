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
	"net"
	"runtime/debug"
	"testing"

	"mosn.io/mosn/pkg/module/http2"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
)

func TestServerCodec(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestServerCodec error: %v %s", r, string(debug.Stack()))
		}
	}()

	testAddr := "127.0.0.1:11234"
	rawc, err := net.Dial("tcp", testAddr)
	if err != nil {
		t.Logf("net.Dial error %v", err)
		return
	}

	c := network.NewServerConnection(context.Background(), rawc, nil)

	buf := buffer.NewIoBufferString(http2.ClientPreface)
	if buf == nil {
		t.Error("New buffer failed.")
	}

	sc := serverCodec{sc: http2.NewServerConn(c)}
	if _, err := sc.Decode(context.Background(), buf); err != nil {
		t.Errorf("Decode failed：%v", err)
	}

	stream := http2.MStream{}
	if _, err := sc.Encode(context.Background(), stream); err != nil {
		t.Errorf("Encode failed：%v", err)
	}

	if p := sc.Name(); p != protocol.HTTP2 {
		t.Errorf("get sc Name failed，want:%v but got: %v", protocol.HTTP2, p)
	}
}

func TestClientCodec(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestClientCodec error: %v %s", r, string(debug.Stack()))
		}
	}()

	testAddr := "127.0.0.1:11234"
	rawc, err := net.Dial("tcp", testAddr)
	if err != nil {
		t.Logf("net.Dial error %v", err)
		return
	}

	c := network.NewServerConnection(context.Background(), rawc, nil)

	buf := buffer.NewIoBufferString(http2.ClientPreface)
	if buf == nil {
		t.Error("New buffer failed.")
	}

	cc := clientCodec{cc: http2.NewClientConn(c)}
	if _, err := cc.Decode(context.Background(), buf); err != nil {
		t.Errorf("Decode failed：%v", err)
	}

	stream := http2.MClientStream{}
	if _, err := cc.Encode(context.Background(), stream); err != nil {
		t.Errorf("Encode failed：%v", err)
	}

	if p := cc.Name(); p != protocol.HTTP2 {
		t.Errorf("get cc Name failed，want:%v but got: %v", protocol.HTTP2, p)
	}
}
