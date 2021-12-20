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

package http

import (
	"github.com/valyala/fasthttp"
	"mosn.io/api"
)

type Code uint32

const (
	Continue Code = 100
	OK            = 200

	Created                     = 201
	Accepted                    = 202
	NonAuthoritativeInformation = 203
	NoContent                   = 204
	ResetContent                = 205
	PartialContent              = 206
	MultiStatus                 = 207
	AlreadyReported             = 208
	IMUsed                      = 226

	MultipleChoices   = 300
	MovedPermanently  = 301
	Found             = 302
	SeeOther          = 303
	NotModified       = 304
	UseProxy          = 305
	TemporaryRedirect = 307
	PermanentRedirect = 308

	BadRequest                  = 400
	Unauthorized                = 401
	PaymentRequired             = 402
	Forbidden                   = 403
	NotFound                    = 404
	MethodNotAllowed            = 405
	NotAcceptable               = 406
	ProxyAuthenticationRequired = 407
	RequestTimeout              = 408
	Conflict                    = 409
	Gone                        = 410
	LengthRequired              = 411
	PreconditionFailed          = 412
	PayloadTooLarge             = 413
	URITooLong                  = 414
	UnsupportedMediaType        = 415
	RangeNotSatisfiable         = 416
	ExpectationFailed           = 417
	MisdirectedRequest          = 421
	UnprocessableEntity         = 422
	Locked                      = 423
	FailedDependency            = 424
	UpgradeRequired             = 426
	PreconditionRequired        = 428
	TooManyRequests             = 429
	RequestHeaderFieldsTooLarge = 431

	InternalServerError           = 500
	NotImplemented                = 501
	BadGateway                    = 502
	ServiceUnavailable            = 503
	GatewayTimeout                = 504
	HTTPVersionNotSupported       = 505
	VariantAlsoNegotiates         = 506
	InsufficientStorage           = 507
	LoopDetected                  = 508
	NotExtended                   = 510
	NetworkAuthenticationRequired = 511

	PlaceHolder = "-"
)

type RequestHeader struct {
	*fasthttp.RequestHeader
}

// Get value of key
func (h RequestHeader) Get(key string) (string, bool) {
	if result := h.Peek(key); result != nil {
		return string(result), true
	}
	return "", false
}

// Set key-value pair in header map, the previous pair will be replaced if exists
//
// Due to the fact that fasthttp's implementation doesn't have
//correct semantic for Set("key", "") and Peek("key") at the
// first time of usage. We need another way for compensate.
//
// The problem is caused by the func initHeaderKV,
//if the original kv.value is nil, ant input value is also nil,
// then the final kv.value remains nil.
//
// kv.value = append(kv.value[:0], value...)
//
// fasthttp do has the kv entry, but kv.value is nil, so Peek("key") return nil. But we want "" instead.
func (h RequestHeader) Set(key string, value string) {
	if value == "" {
		// Set a placeholder first, so that RequestHeader can get this value after setting an empty value.
		h.RequestHeader.Set(key, PlaceHolder)
	}
	h.RequestHeader.Set(key, value)
}

// Add value for given key.
// Multiple headers with the same key may be added with this function.
// Use Set for setting a single header for the given key.
func (h RequestHeader) Add(key, value string) {
	h.RequestHeader.Add(key, value)
}

// Del delete pair of specified key
func (h RequestHeader) Del(key string) {
	h.RequestHeader.Del(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (h RequestHeader) Range(f func(key, value string) bool) {
	stopped := false
	h.VisitAll(func(key, value []byte) {
		if stopped {
			return
		}
		stopped = !f(string(key), string(value))
	})
}

func (h RequestHeader) Clone() api.HeaderMap {
	cpy := &fasthttp.RequestHeader{}
	h.CopyTo(cpy)
	return RequestHeader{cpy}
}

func (h RequestHeader) ByteSize() (size uint64) {
	h.VisitAll(func(key, value []byte) {
		size += uint64(len(key) + len(value))
	})
	return size
}

type ResponseHeader struct {
	*fasthttp.ResponseHeader
}

// Get value of key
func (h ResponseHeader) Get(key string) (string, bool) {
	if result := h.Peek(key); result != nil {
		return string(result), true
	}
	return "", false
}

// Set key-value pair in header map, the previous pair will be replaced if exists
//
// Due to the fact that fasthttp's implementation doesn't have correct semantic for
//Set("key", "") and Peek("key") at the
// first time of usage. We need another way for compensate.
//
// The problem is caused by the func initHeaderKV, if the original kv.value is nil, ant input value is also nil,
// then the final kv.value remains nil.
//
// kv.value = append(kv.value[:0], value...)
//
// fasthttp do has the kv entry, but kv.value is nil, so Peek("key") return nil. But we want "" instead.
func (h ResponseHeader) Set(key string, value string) {
	if value == "" {
		// Set a placeholder first, so that ResponseHeader can get this value after setting an empty value.
		h.ResponseHeader.Set(key, PlaceHolder)
	}
	h.ResponseHeader.Set(key, value)
}

// Add value for given key.
// Multiple headers with the same key may be added with this function.
// Use Set for setting a single header for the given key.
func (h ResponseHeader) Add(key, value string) {
	h.ResponseHeader.Add(key, value)
}

// Del delete pair of specified key
func (h ResponseHeader) Del(key string) {
	h.ResponseHeader.Del(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (h ResponseHeader) Range(f func(key, value string) bool) {
	stopped := false
	h.VisitAll(func(key, value []byte) {
		if stopped {
			return
		}
		stopped = !f(string(key), string(value))
	})
}

func (h ResponseHeader) Clone() api.HeaderMap {
	cpy := &fasthttp.ResponseHeader{}
	h.CopyTo(cpy)
	return ResponseHeader{cpy}
}

func (h ResponseHeader) ByteSize() (size uint64) {
	h.VisitAll(func(key, value []byte) {
		size += uint64(len(key) + len(value))
	})
	return size
}
