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

package echo

import (
	"context"
	"encoding/json"
	"testing"

	"mosn.io/mosn/pkg/networkextention/l7/stream"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/proxy"
)

func TestCreateEchoFilterFactory(t *testing.T) {
	m := map[string]interface{}{}
	if f, err := CreateEchoFilterFactory(m); f == nil || err != nil {
		t.Errorf("CreateEchoFilterFactory failed: %v.", err)
	}
}

func TestEchoNewStreamFilter(t *testing.T) {
	data, err := mockConfig()
	if err != nil {
		t.Fatalf("mock echo config failed: %v", err)
	}

	fa, err := CreateEchoFilterFactory(data)
	if err != nil {
		t.Fatalf("CreateEchoFilterFactory failed: %v", err)
	}

	f := NewEchoFilter(fa.(*FilterConfigFactory))
	sm := stream.CreateActiveStream(context.TODO())
	f.SetReceiveFilterHandler(&sm)
	f.SetSenderFilterHandler(&sm)

	reqHeaders := protocol.CommonHeader(map[string]string{})

	f.OnReceive(context.TODO(), reqHeaders, nil, nil)
	want := "hello world"
	if body := f.senderHandler.GetResponseData(); body == nil || body.String() != want {
		t.Errorf("echo failed: got %+v want %v", body, want)
	}

	headers := f.senderHandler.GetResponseHeaders()
	if headers == nil {
		t.Errorf("echo failed: got response headers %v ", headers)
	}

	want = "test1"
	if v, ok := headers.Get(want); !ok || v != want {
		t.Errorf("echo failed: got %v  want %v", v, want)
	}
}

func BenchmarkEcho(b *testing.B) {
	data, err := mockConfig()
	if err != nil {
		b.Fatalf("mock echo config failed: %v", err)
	}

	fa, err := CreateEchoFilterFactory(data)
	if err != nil {
		b.Fatalf("CreateEchoFilterFactory failed: %v", err)
	}

	f := NewEchoFilter(fa.(*FilterConfigFactory))
	sm := stream.CreateActiveStream(context.TODO())
	f.SetReceiveFilterHandler(&sm)
	f.SetSenderFilterHandler(&sm)

	want := "hello world"
	for i := 0; i < b.N; i++ {
		reqHeaders := protocol.CommonHeader(map[string]string{})
		f.OnReceive(context.TODO(), reqHeaders, nil, nil)
		if body := f.senderHandler.GetResponseData(); body == nil || body.String() != want {
			b.Errorf("echo failed: got %+v want %v", body, want)
		}

	}
}

func mockConfig() (map[string]interface{}, error) {
	mockConfig := `{
                              "status": 200,
			      "header_list": {"test1":"test1", "test2":"test2"},
	                      "body": "hello world"
                       }`
	data := map[string]interface{}{}
	err := json.Unmarshal([]byte(mockConfig), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
