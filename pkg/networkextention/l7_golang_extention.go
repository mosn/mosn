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
#include<stdint.h>
typedef struct {
    const char **headers;
    const char **trailers;
    const char *req_body[2];
    uint64_t cid;
    uint64_t sid;
}Request;

typedef struct {
    const char **headers;
    const char **trailers;
    char *resp_body[2];
    int   status;
    int   direct_response;
}Response;
*/
import "C"

import (
	"context"
	"reflect"
	"runtime/debug"
	"unsafe"

	"mosn.io/api"

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

	hLen := 0
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
func initReqAndResp(req *C.Request, resp *C.Response, headerLen C.size_t, trailerLen C.size_t) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[golang_extention] initReqAndResp() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	// headers -> |key-data|key-len|val-data|val-len|null|
	req.headers = (**C.char)(C.calloc(C.size_t(4*headerLen+1), C.size_t(pointerSize)))
	req.trailers = (**C.char)(C.calloc(C.size_t(4*trailerLen+1), C.size_t(pointerSize)))
	// req_body[0] save request body pointer, req_body[1] save request body length
	req.req_body[0] = nil
	req.req_body[1] = nil
	// resp_body[0] save response body pointer, resp_body[1] save response body length
	resp.resp_body[0] = nil
	resp.resp_body[1] = nil
	resp.headers = nil
	resp.trailers = nil
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

	// free trailer
	freeCharPointer(req.trailers)
	freeCharPointerArray(resp.trailers)

	// free body when direct response
	if resp.resp_body[0] != nil {
		C.free(unsafe.Pointer(resp.resp_body[0]))
	}
}

func createStreamHeaderorTrailer(headers **C.char) stream.Headers {
	header := make(map[string]string)
	update := make(map[string]string)
	if headers != nil {
		rangeCstring(headers, func(k, v string) bool {
			header[k] = v
			return true
		})
	}

	return stream.Headers{
		CommonHeader: protocol.CommonHeader(header),
		Update:       update,
	}
}

//export runReceiveStreamFilter
func runReceiveStreamFilter(req C.Request) C.Response {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[golang_extention] runReceiveStreamFilter() panic %v\n%s", r, string(debug.Stack()))
		}
	}()

	// headers
	requestHeader := createStreamHeaderorTrailer(req.headers)

	// body
	var data string
	if req.req_body[0] != nil && req.req_body[1] != nil {
		data = dataToString(req.req_body[0], req.req_body[1])
	}

	// trailers
	requestTrailer := createStreamHeaderorTrailer(req.trailers)

	// TODO init filter manager after support stream id
	// create stream
	sm := stream.CreateActiveStream(context.TODO())
	sm.InitResuestStream(&requestHeader, buffer.NewIoBufferString(data), &requestTrailer)
	sm.SetConnectionID(uint64(req.cid))
	sm.SetStreamID(uint64(req.sid))

	fm := stream.CreateStreamFilter(context.TODO(), stream.DefaultFilterChainName)
	fm.SetReceiveFilterHandler(&sm)

	fm.RunReceiverFilter(context.TODO(), api.BeforeRoute, sm.GetRequestHeaders(), sm.GetRequestData(), sm.GetRequestTrailers(), nil)
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

	// headers
	requestHeader := createStreamHeaderorTrailer(resp.headers)

	// TODO support GetorCreateActive by stream id
	sm := stream.CreateActiveStream(context.TODO())
	var data string
	if resp.req_body[0] != nil && resp.req_body[1] != nil {
		data = dataToString(resp.req_body[0], resp.req_body[1])
	}

	// trailers
	requestTrailer := createStreamHeaderorTrailer(resp.trailers)

	sm.InitResponseStream(&requestHeader, buffer.NewIoBufferString(data), &requestTrailer)

	fm := stream.CreateStreamFilter(context.TODO(), stream.DefaultFilterChainName)
	fm.SetSenderFilterHandler(&sm)
	fm.RunSenderFilter(context.TODO(), api.BeforeSend, sm.GetResponseHeaders(), sm.GetResponseData(), sm.GetResponseTrailers(), nil)
	stream.DestoryStreamFilter(fm)

	return convertCResponse(sm)
}

func convertCRequest(sm stream.ActiveStream) C.Response {
	var resp C.Response

	if sm.IsDirectResponse() {
		resp.status = C.int(sm.GetResponseCode())
		resp.direct_response = 1
		if data := sm.GetResponseData(); data != nil && len(data.String()) != 0 {
			resp.resp_body[0] = C.CString(data.String())
			resp.resp_body[1] = (*C.char)(unsafe.Pointer(uintptr(len(data.String()))))
		}
		rangeMapToChar(sm.GetResponseHeaders(), &resp.headers)
		rangeMapToChar(sm.GetResponseTrailers(), &resp.trailers)
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
		if data := sm.GetResponseData(); data != nil && len(data.String()) != 0 {
			resp.resp_body[0] = C.CString(data.String())
			resp.resp_body[1] = (*C.char)(unsafe.Pointer(uintptr(len(data.String()))))
		}
		rangeMapToChar(sm.GetResponseHeaders(), &resp.headers)
		rangeMapToChar(sm.GetResponseTrailers(), &resp.trailers)
		return resp
	}

	rangeMapToChar(sm.GetResponseUpdatedHeaders(), &resp.headers)

	return resp
}
