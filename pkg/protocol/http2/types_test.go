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
	"net/http"
	"testing"
)

const MosnHeaderHostKey = "Mosn-Header-Host"
const MosnHeaderContentTypeKey = "Content-Type"
const MosnHeaderEmptyKey = "Mosn-Empty"

func TestRequestHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestReqHeader error: %v", r)
		}
	}()

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	header := NewReqHeader(req)

	header.Set(MosnHeaderHostKey, "test")
	if v, ok := header.Get(MosnHeaderHostKey); !ok || v != "test" {
		t.Error("Get header failed.")
	}

	header.Del(MosnHeaderHostKey)
	if _, ok := header.Get(MosnHeaderHostKey); ok {
		t.Error("Del header failed.")
	}

	// test clone header
	header.Set(MosnHeaderHostKey, "test")
	h2 := header.Clone()
	if h2 == nil {
		t.Error("Clone header failed.")
	}
	if v, ok := header.Get(MosnHeaderHostKey); !ok || v != "test" {
		t.Error("Clone header failed.")
	}

	// test ByteSize
	if l := h2.ByteSize(); l != uint64(len(MosnHeaderHostKey)+len("test")) {
		t.Errorf("get ByteSize failed got: %d want:%d", l, len(MosnHeaderHostKey)+len("test"))
	}

	// test range header
	h2.Range(func(key, value string) bool {
		if key != MosnHeaderHostKey || value != "test" {
			t.Errorf("Range header failed: %v, %v", key, value)
			return false
		}
		return true
	})

	// test set empty header
	h2.Set(MosnHeaderEmptyKey, "")
	if v, ok := header.Get(MosnHeaderEmptyKey); ok || v != "" {
		t.Errorf("Set empty header failed: %v %v", v, ok)
	}

}

func TestResponseHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestResponseHeader error: %v", r)
		}
	}()

	header := NewRspHeader(&http.Response{Header: make(http.Header)})

	header.Set(MosnHeaderContentTypeKey, "test")
	if v, ok := header.Get(MosnHeaderContentTypeKey); !ok || v != "test" {
		t.Error("Get header failed.")
	}

	// test clone header
	header.Set(MosnHeaderContentTypeKey, "test")
	h2 := header.Clone()
	if h2 == nil {
		t.Error("Clone header failed.")
	}
	if v, ok := header.Get(MosnHeaderContentTypeKey); !ok || v != "test" {
		t.Error("Clone header failed.")
	}

	// test ByteSize
	if l := h2.ByteSize(); l != uint64(len(MosnHeaderContentTypeKey)+len("test")) {
		t.Errorf("get ByteSize failed got: %d want:%d", l, len(MosnHeaderContentTypeKey)+len("test"))
	}

	// test range header
	h2.Range(func(key, value string) bool {
		if key != MosnHeaderContentTypeKey || value != "test" {
			t.Error("Range header failed.")
			return false
		}
		return true
	})

	// test set empty header
	h2.Set(MosnHeaderHostKey, "")
	if _, ok := header.Get(MosnHeaderHostKey); ok {
		t.Error("Set empty header failed.")
	}
}
