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

package trace

import (
	"context"
	"testing"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/proxy"
)

func TestCreateTraceFilterFactory(t *testing.T) {
	m := map[string]interface{}{}
	if _, err := CreateTraceFilterFactory(m); err != nil {
		t.Errorf("CreateTraceFilterFactory failed: %v.", err)
	}
}

func TestGzipNewStreamFilter(t *testing.T) {
	cfg := &v2.StreamTrace{
		TracerTraceidArg:    "traceid_arg",
		TracerTraceidHeader: "traceid_header",
		TracerRpcidHeader:   "rpcid_header",
	}

	f := NewTraceFilter(context.TODO(), cfg)
	argtid := "1111111111111111111111111"
	// test trace id from arg
	reqHeaders := protocol.CommonHeader(map[string]string{
		pathPrefix: "/p?" + f.trace.TracerTraceidArg + "=" + argtid,
	})

	f.OnReceive(context.TODO(), reqHeaders, nil, nil)

	if v, ok := reqHeaders.Get(f.trace.TracerTraceidHeader); !ok || v != argtid {
		t.Error("get trace id failed form arg.")
	}

	// test trace id from header
	headertid := "1111111111111111111111112"
	reqHeaders = protocol.CommonHeader(map[string]string{
		f.trace.TracerTraceidHeader: headertid,
	})
	f.OnReceive(context.TODO(), reqHeaders, nil, nil)
	if v, ok := reqHeaders.Get(f.trace.TracerTraceidHeader); !ok || v != headertid {
		t.Error("get trace id failed form arg.")
	}

	// test trace id from arg or header
	reqHeaders = protocol.CommonHeader(map[string]string{
		f.trace.TracerTraceidHeader: headertid,
		pathPrefix:                  "/p?" + f.trace.TracerTraceidArg + "=" + argtid,
	})
	f.OnReceive(context.TODO(), reqHeaders, nil, nil)
	if v, ok := reqHeaders.Get(f.trace.TracerTraceidHeader); !ok || v != argtid {
		t.Error("get trace id failed form arg when both arg and headerid exist, it is preferred to obtain them from argid.")
	}

	// test invalid argid
	invalidArgid := "h1111111111111111111111111"
	reqHeaders = protocol.CommonHeader(map[string]string{
		f.trace.TracerTraceidHeader: headertid,
		pathPrefix:                  "/p?" + f.trace.TracerTraceidArg + "=" + invalidArgid,
	})
	f.OnReceive(context.TODO(), reqHeaders, nil, nil)
	if v, ok := reqHeaders.Get(f.trace.TracerTraceidHeader); !ok || v != headertid {
		t.Error("get trace id failed form header when argid is invalid.")
	}

	// test the scenario when rpcid already exists
	rpcid := "0.1"
	reqHeaders = protocol.CommonHeader(map[string]string{
		f.trace.TracerRpcidHeader: rpcid,
	})
	f.OnReceive(context.TODO(), reqHeaders, nil, nil)
	if v, ok := reqHeaders.Get(f.trace.TracerRpcidHeader); !ok || v != rpcid+RpcIdChild {
		t.Error("get rpcid failed.")
	}

	// test rpcid is ""
	rpcid = ""
	reqHeaders = protocol.CommonHeader(map[string]string{
		f.trace.TracerRpcidHeader: rpcid,
	})
	f.OnReceive(context.TODO(), reqHeaders, nil, nil)
	if v, ok := reqHeaders.Get(f.trace.TracerRpcidHeader); !ok || v != RpcIdChild {
		t.Error("get rpcid failed.")
	}
}

func BenchmarkTrace(b *testing.B) {
	cfg := &v2.StreamTrace{
		TracerTraceidArg:    "traceid_arg",
		TracerTraceidHeader: "traceid_header",
		TracerRpcidHeader:   "rpcid_header",
	}

	f := NewTraceFilter(context.TODO(), cfg)

	for i := 0; i < b.N; i++ {
		reqHeaders := protocol.CommonHeader(map[string]string{})
		f.OnReceive(context.TODO(), reqHeaders, nil, nil)
		if _, ok := reqHeaders.Get(f.trace.TracerTraceidHeader); !ok {
			b.Error("get trace id failed.")
		}

		if _, ok := reqHeaders.Get(f.trace.TracerRpcidHeader); !ok {
			b.Error("get rpc id failed.")
		}
	}
}
