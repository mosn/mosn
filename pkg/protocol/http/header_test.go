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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
)

func TestRequestHeader_Add(t *testing.T) {
	header := RequestHeader{&fasthttp.RequestHeader{}, nil}
	header.Add("test-multiple", "value-one")
	header.Add("test-multiple", "value-two")

	// assert Peek results
	val := header.Peek("test-multiple")
	if string(val) != "value-one" {
		t.Errorf("RequestHeader.Get return not expected")
	}

	// assert output results
	output := header.String()
	if !strings.Contains(output, "value-one") || !strings.Contains(output, "value-two") {
		t.Errorf("RequestHeader.String not contains all header values")
	}
}

func TestResponseHeader_Add(t *testing.T) {
	header := ResponseHeader{&fasthttp.ResponseHeader{}, nil}
	header.Add("test-multiple", "value-one")
	header.Add("test-multiple", "value-two")

	// assert Peek results
	val := header.Peek("test-multiple")
	if string(val) != "value-one" {
		t.Errorf("ResponseHeader.Get return not expected")
	}

	// assert output results
	output := header.String()
	if !strings.Contains(output, "value-one") || !strings.Contains(output, "value-two") {
		t.Errorf("ResponseHeader.String not contains all header values")
	}
}

func TestHeaderSetEmptyKey(t *testing.T) {
	tests := []struct {
		name   string
		header api.HeaderMap
	}{
		{
			name:   "request header",
			header: &ResponseHeader{&fasthttp.ResponseHeader{}, nil},
		},
		{
			name:   "response header",
			header: &RequestHeader{&fasthttp.RequestHeader{}, nil},
		},
	}

	key := "foobar"
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.header.Set(key, "")

			// value is "", key exists
			value, exists := tc.header.Get(key)
			assert.True(t, exists)
			assert.Equal(t, "", value)

			tc.header.Del(key)
			_, exists = tc.header.Get(key)
			assert.False(t, exists)
		})
	}
}
