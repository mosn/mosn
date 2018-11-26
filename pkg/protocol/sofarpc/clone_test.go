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

package sofarpc

import (
	"reflect"
	"testing"
)

func TestReqCommandClone(t *testing.T) {
	headers := map[string]string{"service": "test"}
	// test headers, the others ignore
	reqCmd := &BoltRequestCommand{
		RequestHeader: headers,
	}
	reqClone := reqCmd.Clone()
	if !reflect.DeepEqual(reqCmd, reqClone) {
		t.Error("clone data not equal")
	}
	// modify clone header
	reqClone.Set("testclone", "test")
	reqClone.Set("service", "clone")
	// new header is setted
	if v, ok := reqClone.Get("service"); !ok || v != "clone" {
		t.Error("clone header is not setted")
	}
	if v, ok := reqClone.Get("testclone"); !ok || v != "test" {
		t.Error("clone header is not setted")
	}
	// original header is not effected
	if v, ok := reqCmd.Get("service"); !ok || v != "test" {
		t.Error("original header is chaned")
	}
	if _, ok := reqCmd.Get("testclone"); ok {
		t.Error("original header is chaned")
	}
}

func TestRespCommandClone(t *testing.T) {
	headers := map[string]string{"service": "test"}
	// test headers, the others ignore
	respCmd := &BoltResponseCommand{
		ResponseHeader: headers,
	}
	respClone := respCmd.Clone()
	if !reflect.DeepEqual(respCmd, respClone) {
		t.Error("clone data not equal")
	}
	// modify clone header
	respClone.Set("testclone", "test")
	respClone.Set("service", "clone")
	// new header is setted
	if v, ok := respClone.Get("service"); !ok || v != "clone" {
		t.Error("clone header is not setted")
	}
	if v, ok := respClone.Get("testclone"); !ok || v != "test" {
		t.Error("clone header is not setted")
	}
	// original header is not effected
	if v, ok := respCmd.Get("service"); !ok || v != "test" {
		t.Error("original header is chaned")
	}
	if _, ok := respCmd.Get("testclone"); ok {
		t.Error("original header is chaned")
	}
}
