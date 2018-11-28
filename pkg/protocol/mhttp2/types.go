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

package mhttp2

import (
	"net/http"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type HeaderMap struct {
	H http.Header
}

type ReqHeader struct {
	*HeaderMap
	Req *http.Request
}

type RspHeader struct {
	*HeaderMap
	Rsp *http.Response
}

func NewHeaderMap(header http.Header) *HeaderMap {
	h := new(HeaderMap)
	h.H = header
	return h
}

func NewReqHeader(req *http.Request) *ReqHeader {
	h := new(ReqHeader)
	h.Req = req
	h.HeaderMap = NewHeaderMap(h.Req.Header)
	return h
}

func NewRspHeader(rsp *http.Response) *RspHeader {
	h := new(RspHeader)
	h.Rsp = rsp
	h.HeaderMap = NewHeaderMap(h.Rsp.Header)
	return h
}

// Get value of key
func (h *HeaderMap) Get(key string) (value string, ok bool) {
	value = h.H.Get(key)

	return value, true
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (h *HeaderMap) Set(key string, value string) {
	h.H.Set(key, value)
}

// Del delete pair of specified key
func (h *HeaderMap) Del(key string) {
	h.H.Del(key)
}

func (h *HeaderMap) Clone() types.HeaderMap {
	header := h.H
	h2 := make(http.Header, len(header))
	for k, vv := range header {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}

	return NewHeaderMap(h2)
}

func (h *ReqHeader) Clone() types.HeaderMap {
	h2 := new(ReqHeader)
	h2.HeaderMap = h.HeaderMap.Clone().(*HeaderMap)
	h2.Req = new(http.Request)
	*h2.Req = *h.Req
	h2.Req.Header = h2.HeaderMap.H
	return h2
}

func (h *ReqHeader) Get(key string) (string, bool) {
	if len(key) > 0 && key[0] == ':' {
		switch key {
		case ":authority":
			return h.Req.Host, true
		case ":path":
			return h.Req.RequestURI, true
		case ":method":
			return h.Req.Method, true
		default:
			return "", false
		}
	}
	return h.HeaderMap.Get(key)
}

func (h *RspHeader) Clone() types.HeaderMap {
	h2 := new(RspHeader)
	h2.HeaderMap = h.HeaderMap.Clone().(*HeaderMap)
	h2.Rsp = new(http.Response)
	*h2.Rsp = *h.Rsp
	h2.Rsp.Header = h2.HeaderMap.H
	return h2
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (h HeaderMap) Range(f func(key, value string) bool) {
	for k, v := range h.H {
		// stop if f return false
		if !f(k, v[0]) {
			break
		}
	}
}

func (h HeaderMap) ByteSize() uint64 {
	var size uint64

	for k, v := range h.H {
		size += uint64(len(k) + len(v[0]))
	}
	return size
}

func EncodeHeader(in map[string]string) (header http.Header) {
	header = http.Header((make(map[string][]string, len(in))))
	for k, v := range in {
		header.Add(k, v)
	}
	return header
}

func DecodeHeader(header types.HeaderMap) types.HeaderMap {
	var in http.Header
	switch h := header.(type) {
	case *ReqHeader:
		in = h.H
	case *RspHeader:
		in = h.H
	case *HeaderMap:
		in = h.H
	default:
		return nil
	}

	out := make(map[string]string, len(in))
	for k, v := range in {
		out[strings.ToLower(k)] = strings.Join(v, ",")
	}
	return protocol.CommonHeader(out)
}
