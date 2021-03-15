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
	"testing"

	"github.com/valyala/fasthttp"
)

const testHeaderHostKey = "Mosn-Header-Host"
const testHeaderContentTypeKey = "Content-Type"
const testHeaderEmptyKey = "Mosn-Empty"

func TestRequestHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestCommonHeader error: %v", r)
		}
	}()

	header := RequestHeader{&fasthttp.RequestHeader{}}

	header.Set(testHeaderHostKey, "test")
	if v, ok := header.Get(testHeaderHostKey); !ok || v != "test" {
		t.Error("Get header failed.")
	}

	header.Del(testHeaderHostKey)
	if _, ok := header.Get(testHeaderHostKey); ok {
		t.Error("Del header failed.")
	}

	// test clone header
	header.Set(testHeaderHostKey, "test")
	h2 := header.Clone()
	if h2 == nil {
		t.Error("Clone header failed.")
	}
	if v, ok := header.Get(testHeaderHostKey); !ok || v != "test" {
		t.Error("Clone header failed.")
	}

	// test ByteSize
	if l := h2.ByteSize(); l != uint64(len(testHeaderHostKey)+len("test")) {
		t.Errorf("get ByteSize failed got: %d want:%d", l, len(testHeaderHostKey)+len("test"))
	}

	// test range header
	h2.Range(func(key, value string) bool {
		if key != testHeaderHostKey || value != "test" {
			t.Errorf("Range header failed: %v, %v", key, value)
			return false
		}
		return true
	})

}

func TestEmptyValueForRequestHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestCommonHeader error: %v", r)
		}
	}()
	header := RequestHeader{&fasthttp.RequestHeader{}}

	header.Set(testHeaderEmptyKey, "")
	if v, ok := header.Get(testHeaderEmptyKey); !ok || v != "" {
		t.Errorf("Set empty header failed: %v %v", v, ok)
	}
}

func TestResponseHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestCommonHeader error: %v", r)
		}
	}()

	header := ResponseHeader{&fasthttp.ResponseHeader{}}

	header.Set(testHeaderContentTypeKey, "test")
	if v, ok := header.Get(testHeaderContentTypeKey); !ok || v != "test" {
		t.Error("Get header failed.")
	}

	// test clone header
	header.Set(testHeaderContentTypeKey, "test")
	h2 := header.Clone()
	if h2 == nil {
		t.Error("Clone header failed.")
	}
	if v, ok := header.Get(testHeaderContentTypeKey); !ok || v != "test" {
		t.Error("Clone header failed.")
	}

	// test ByteSize
	if l := h2.ByteSize(); l != uint64(len(testHeaderContentTypeKey)+len("test")) {
		t.Errorf("get ByteSize failed got: %d want:%d", l, len(testHeaderContentTypeKey)+len("test"))
	}

	// test range header
	h2.Range(func(key, value string) bool {
		if key != testHeaderContentTypeKey || value != "test" {
			t.Error("Range header failed.")
			return false
		}
		return true
	})

	// test set empty header
	h2.Set(testHeaderHostKey, "")
	if _, ok := header.Get(testHeaderHostKey); ok {
		t.Error("Set empty header failed.")
	}
}

func TestEmptyValueForResponseHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestCommonHeader error: %v", r)
		}
	}()
	header := ResponseHeader{&fasthttp.ResponseHeader{}}

	header.Set(testHeaderEmptyKey, "")
	if v, ok := header.Get(testHeaderEmptyKey); !ok || v != "" {
		t.Errorf("Set empty header failed: %v %v", v, ok)
	}
}
