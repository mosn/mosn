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

package main

/*
#include <stdlib.h>
#include <string.h>
typedef struct {
    const char **headers;
    char *req_body;
}Request;

typedef struct {
    const char **headers;
    char *resp_body;
    int   status;
    int   direct_response;
}Response;
*/
import "C"

import (
	"context"
	"mosn.io/api"
	"reflect"
	"runtime/debug"
	"unsafe"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/networkextention/l7/stream"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

// sizeof pointer offset
var pointerSize = unsafe.Sizeof(uintptr(0))

func dataToString(data *C.char, lenPointer *C.char) string {
	var s0 string
	var s0Hdr = (*reflect.StringHeader)(unsafe.Pointer(&s0))
	s0Hdr.Data = uintptr(unsafe.Pointer(data))
	s0Hdr.Len = int(uintptr(unsafe.Pointer(lenPointer)))
	return s0
}

func rangeCstring(headers **C.char, f func(k, v string) bool) {
	for *headers != nil {
		kData := headers
		kLen := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(headers)) + pointerSize))
		vData := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(headers)) + 2*pointerSize))
		vLen := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(headers)) + 3*pointerSize))
		if !f(dataToString(*kData, *kLen), dataToString(*vData, *vLen)) {
			break
		}

		headers = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(headers)) + (4 * pointerSize)))
	}
}

func rangeMapToChar(header types.HeaderMap, dstHeaders ***C.char) {
	var hLen int
	header.Range(func(k, v string) bool {
		hLen++
		return true
	})

	*dstHeaders = (**C.char)(C.calloc(C.size_t(4*hLen+1), C.size_t(pointerSize)))

	tempHeader := *dstHeaders
	header.Range(func(k, v string) bool {
		// tempHeaders -> |key-data|key-len|val-data|val-len|null|
		*tempHeader = C.CString(k)
		tempHeader = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(tempHeader)) + pointerSize))

		*tempHeader = (*C.char)(unsafe.Pointer(uintptr(len(k))))
		tempHeader = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(tempHeader)) + pointerSize))

		*tempHeader = C.CString(v)
		tempHeader = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(tempHeader)) + pointerSize))

		*tempHeader = (*C.char)(unsafe.Pointer(uintptr(len(v))))
		tempHeader = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(tempHeader)) + pointerSize))
		return true
	})
}

func freeCharPointerArray(p **C.char) {
	if p == nil {
		return
	}

	temp := p
	for *temp != nil {
		C.free(unsafe.Pointer(*temp))
		// |key-data|key-len|val-data|val-len|null| -> |heap mem|int|heap mem|int|null|
		temp = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(temp)) + 2*pointerSize))
	}

	freeCharPointer(p)
}

func freeCharPointer(p **C.char) {
	if p == nil {
		return
	}

	C.free(unsafe.Pointer(p))
}

//export initReqAndResp
func initReqAndResp(req *C.Request, resp *C.Response, len C.size_t) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[golang_extention] initReqAndResp() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	// headers -> |key-data|key-len|val-data|val-len|null|
	req.headers = (**C.char)(C.calloc(C.size_t(4*len+1), C.size_t(pointerSize)))
	req.req_body = nil
	resp.resp_body = nil
	resp.headers = nil
	resp.direct_response = 0
}

//export freeReqAndResp
func freeReqAndResp(req *C.Request, resp *C.Response) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[golang_extention] freeReqAndResp() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	if req == nil || resp == nil {
		return
	}

	// free header
	freeCharPointer(req.headers)
	freeCharPointerArray(resp.headers)

	// free body when direct response
	if resp.resp_body != nil {
		C.free(unsafe.Pointer(resp.resp_body))
	}
}

//export runReceiveStreamFilter
func runReceiveStreamFilter(req C.Request) C.Response {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[golang_extention] runReceiveStreamFilter() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	header := make(map[string]string)
	update := make(map[string]string)
	if req.headers != nil {
		rangeCstring(req.headers, func(k, v string) bool {
			header[k] = v
			return true
		})
	}

	fm := stream.CreateStreamFilter(context.TODO(), stream.DefaultFilterChainName)
	requestHeader := stream.Headers{
		CommonHeader: protocol.CommonHeader(header),
		Update:       update,
	}

	sm := stream.CreateActiveStream(context.TODO())
	var data string
	if req.req_body != nil {
		data = dataToString(req.req_body, (*C.char)(unsafe.Pointer(uintptr(C.strlen(req.req_body)))))
	}

	// TODO suppport trailer
	sm.InitResuestStream(&requestHeader, buffer.NewIoBufferString(data), nil)

	fm.SetReceiveFilterHandler(&sm)

	fm.RunReceiverFilter(context.TODO(), api.BeforeRoute, &requestHeader, nil, nil, nil)
	sm.SetCurrentReveivePhase(api.BeforeRoute)

	stream.DestoryStreamFilter(fm)

	return convertCRequest(sm)
}

//export runSendStreamFilter
func runSendStreamFilter(resp C.Request) C.Response {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[golang_extention] runSendStreamFilter() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	header := make(map[string]string)
	update := make(map[string]string)
	if resp.headers != nil {
		rangeCstring(resp.headers, func(k, v string) bool {
			header[k] = v
			return true
		})
	}

	// TODO reuse
	requestHeader := stream.Headers{
		CommonHeader: protocol.CommonHeader(header),
		Update:       update,
	}

	// TODO support GetorCreateActive by stream id
	sm := stream.CreateActiveStream(context.TODO())
	var data string
	if resp.req_body != nil {
		data = dataToString(resp.req_body, (*C.char)(unsafe.Pointer(uintptr(C.strlen(resp.req_body)))))
	}

	// TODO suppport trailer
	sm.InitResponseStream(&requestHeader, buffer.NewIoBufferString(data), nil)

	fm := stream.CreateStreamFilter(context.TODO(), stream.DefaultFilterChainName)
	fm.SetSenderFilterHandler(&sm)
	fm.RunSenderFilter(context.TODO(), api.BeforeSend, &requestHeader, nil, nil, nil)
	stream.DestoryStreamFilter(fm)

	return convertCResponse(sm)
}

func convertCRequest(sm stream.ActiveStream) C.Response {
	var resp C.Response

	if sm.IsDirectResponse() {
		resp.status = C.int(sm.GetResponseCode())
		resp.direct_response = 1
		if data := sm.GetResponseData(); data != nil {
			resp.resp_body = C.CString(data.String())
		}
		rangeMapToChar(sm.GetResponseHeaders(), &resp.headers)
		return resp
	}

	rangeMapToChar(sm.GetRequestUpdatedHeaders(), &resp.headers)

	return resp
}

func convertCResponse(sm stream.ActiveStream) C.Response {
	var resp C.Response

	if sm.IsDirectResponse() {
		resp.status = C.int(sm.GetResponseCode())
		resp.direct_response = 1
		if data := sm.GetResponseData(); data != nil {
			resp.resp_body = C.CString(data.String())
		}
		rangeMapToChar(sm.GetResponseHeaders(), &resp.headers)
		return resp
	}

	rangeMapToChar(sm.GetResponseUpdatedHeaders(), &resp.headers)

	return resp
}
