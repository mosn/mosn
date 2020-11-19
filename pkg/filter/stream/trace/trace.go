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
	"errors"
	"net/url"
	"runtime/debug"
	"unsafe"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

type TraceFilter struct {
	context context.Context // variable scenarios need to be used.
	trace   *v2.StreamTrace
}

// NewTraceFilter used to create new trace filter
func NewTraceFilter(ctx context.Context, trace *v2.StreamTrace) *TraceFilter {
	filter := &TraceFilter{
		context: ctx,
		trace:   trace,
	}
	return filter
}

func (f *TraceFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	defer func() {
		if r := recover(); r != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [trace] OnReceive() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	if f.trace.Disable {
		return api.StreamFilterContinue
	}

	// check rpc id from header
	var rpcid string
	if v, ok := headers.Get(f.trace.TracerRpcidHeader); !ok {
		rpcid = DefaultRpcId
	} else {
		rpcid = v + RpcIdChild
	}

	headers.Set(f.trace.TracerRpcidHeader, rpcid)

	// check trace id from arg
	if v, ok := headers.Get(pathPrefix); ok && checkArgSizeAndSpell(f.trace, v, f.trace.TracerTraceidArg) {
		traceid, _ := getTraceidFromArg(v, f.trace.TracerTraceidArg)
		headers.Set(f.trace.TracerTraceidHeader, traceid)
		return api.StreamFilterContinue
	}

	// check trace id from header
	if _, ok := headers.Get(f.trace.TracerTraceidHeader); !ok {
		tracer, err := NewTraceIDGenerator(f.trace.TracerIpPart)
		if err != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [trace] OnReceive() generate trace id err: %v", err)
			return api.StreamFilterContinue
		}

		headers.Set(f.trace.TracerTraceidHeader, tracer.String())
	}

	return api.StreamFilterContinue
}

func (f *TraceFilter) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {

	return api.StreamFilterContinue
}

func (f *TraceFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
}

func (f *TraceFilter) OnDestroy() {}

func (f *TraceFilter) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	defer func() {
		if r := recover(); r != nil {
			log.Proxy.Errorf(ctx, "[stream filter] [trace] Log() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	if f.trace.Disable {
		return
	}

	// TODO write format trace log

}

// :path /?trace-id=111111111111111
func getTraceidFromArg(uri, name string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if v := u.Query().Get(name); v != "" {
		return v, nil
	}

	return "", errors.New("not found arg: " + name)
}

func checkArgSizeAndSpell(trace *v2.StreamTrace, uri, name string) bool {
	v, err := getTraceidFromArg(uri, name)
	if err != nil {
		return false
	}

	if trace.TracerIgnoreSize && trace.TracerIgnoreSpellcheck {
		return true
	}

	// check size
	vlen := len(v)
	if !trace.TracerIgnoreSize && ((vlen < TracerTraceIdMinSize) || (vlen > TracerTraceIdMaxSize)) {
		return false
	}

	// check spell

	if !trace.TracerIgnoreSpellcheck && !checkTraceidSpell((&[]byte(v)[0]), uint32(vlen)) {
		return false
	}

	return true
}

/* 1111 1111 1111 1111  1111 1111 1111 1111 */ /* 0xffffffff */
/* ?>=< ;:98 7654 3210  /.-, +*)( '&%$ #"!  */ //
/* 1111 1100 0000 0000  1111 1111 1111 1111 */ /* 0xfc00ffff */
/* _^]\ [ZYX WVUT SRQP  ONML KJIH GFED CBA@ */ //
/* 1111 1111 1111 1111  1111 1111 1111 1111 */ /* 0xffffffff */
/*  ~}| {zyx wvut srqp  onml kjih gfed cba` */ //
/* 1111 1111 1111 1111  1111 1111 1000 0001 */ /* 0xffffff81 */
/* 1111 1111 1111 1111  1111 1111 1111 1111 */ /* 0xffffffff */
/* 1111 1111 1111 1111  1111 1111 1111 1111 */ /* 0xffffffff */
/* 1111 1111 1111 1111  1111 1111 1111 1111 */ /* 0xffffffff */
/* 1111 1111 1111 1111  1111 1111 1111 1111 */ /* 0xffffffff */
func checkTraceidSpell(data *byte, len uint32) bool {
	var invalid []uint32 = []uint32{0xffffffff, 0xfc00ffff, 0xffffffff, 0xffffff81, 0xffffffff, 0xffffffff, 0xffffffff, 0xffffffff}
	var p *byte
	for p = data; int64(uintptr(unsafe.Pointer(p))) < int64(uintptr(unsafe.Pointer(((*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(data)) + (uintptr)(int32(uint32((len))))*unsafe.Sizeof(*data))))))); p = ((*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + (uintptr)(1)*unsafe.Sizeof(*p)))) {
		if uint32((*((*uint32)(func() unsafe.Pointer {
			tempVar := &invalid[0]
			return unsafe.Pointer(uintptr(unsafe.Pointer(tempVar)) + (uintptr)(int32(*p)>>uint64(int32(5)))*unsafe.Sizeof(*tempVar))
		}())) & uint32((uint32(int32(1) << uint64(int32(*p)&int32(0x1f))))))) != 0 {
			return false
		}
	}
	return true
}
