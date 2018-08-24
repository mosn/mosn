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

package subprotocol

import (
	"testing"
)

func Test_SplitRequest_01(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 30}
	rpc := NewRPCHSF()
	reqs := rpc.SplitRequest(msg)
	reqsLen := len(reqs)
	if reqsLen != 0 {
		t.Errorf("%d != 0", reqsLen)
	} else {
		t.Log("split response ok")
	}
}

func Test_SplitRequest_02(t *testing.T) {
	msg := []byte{0x0e, 1, 2, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 30}
	rpc := NewRPCHSF()
	reqs := rpc.SplitRequest(msg)
	reqsLen := len(reqs)
	if reqsLen != 0 {
		t.Errorf("%d != 0", reqsLen)
	} else {
		t.Log("split illegal type msg ok")
	}
}

func Test_SplitRequest_03(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b', 0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x79, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'c', 'd', 0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x77, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'e', 'f'}
	rpc := NewRPCHSF()
	reqs := rpc.SplitRequest(msg)
	reqsLen := len(reqs)
	if reqsLen != 3 {
		t.Errorf("%d != 3", reqsLen)
	} else {
		t.Log("split mulit-request ok")
	}
}

func Test_SplitRequest_04(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b', 0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x79, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'c'}
	rpc := NewRPCHSF()
	reqs := rpc.SplitRequest(msg)
	reqsLen := len(reqs)
	if reqsLen != 1 {
		t.Errorf("%d != 1", reqsLen)
	} else {
		t.Log("split half-baked-request ok")
	}
}

func Test_SplitRequest_05(t *testing.T) {
	msg := []byte{14, 1, 0, 4, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 208, 0, 0, 0, 45, 0, 0, 0, 8, 0, 0, 0, 1, 0, 0, 0, 16, 0, 0, 0, 6, 0, 0, 0, 143, 99, 111, 109, 46, 116, 101, 115, 116, 46, 112, 97, 110, 100, 111, 114, 97, 46, 104, 115, 102, 46, 72, 101, 108, 108, 111, 83, 101, 114, 118, 105, 99, 101, 58, 49, 46, 48, 46, 48, 46, 100, 97, 105, 108, 121, 115, 97, 121, 72, 101, 108, 108, 111, 106, 97, 118, 97, 46, 108, 97, 110, 103, 46, 83, 116, 114, 105, 110, 103, 5, 119, 111, 114, 108, 100, 72, 16, 67, 111, 110, 115, 117, 109, 101, 114, 45, 65, 112, 112, 78, 97, 109, 101, 15, 104, 115, 102, 45, 115, 101, 114, 118, 101, 114, 45, 100, 101, 109, 111, 12, 116, 97, 114, 103, 101, 116, 95, 103, 114, 111, 117, 112, 4, 72, 83, 70, 49, 4, 95, 84, 73, 68, 78, 16, 101, 97, 103, 108, 101, 101, 121, 101, 95, 99, 111, 110, 116, 101, 120, 116, 72, 7, 116, 114, 97, 99, 101, 73, 100, 30, 48, 97, 57, 55, 54, 51, 54, 55, 49, 53, 51, 53, 48, 56, 50, 55, 56, 54, 49, 48, 55, 49, 48, 48, 49, 100, 48, 48, 54, 49, 5, 114, 112, 99, 73, 100, 1, 57, 16, 101, 97, 103, 108, 101, 69, 121, 101, 85, 115, 101, 114, 68, 97, 116, 97, 78, 90, 90}
	rpc := NewRPCHSF()
	reqs := rpc.SplitRequest(msg)
	reqsLen := len(reqs)
	if reqsLen != 1 {
		t.Errorf("%d != 1", reqsLen)
	} else {
		t.Log("split request ok")
	}
}

func Test_SplitRequest_06(t *testing.T) {
	msg := []byte{14, 1, 0, 4, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 208, 0, 0, 0, 45, 0, 0, 0, 8, 0, 0, 0, 1, 0, 0, 0, 16, 0, 0, 0, 6, 0, 0, 0, 143, 99, 111, 109, 46, 116, 101, 115, 116, 46, 112, 97, 110, 100, 111, 114, 97, 46, 104, 115, 102, 46, 72, 101, 108, 108, 111, 83, 101, 114, 118, 105, 99, 101, 58, 49, 46, 48, 46, 48, 46, 100, 97, 105, 108, 121, 115, 97, 121, 72, 101, 108, 108, 111, 106, 97, 118, 97, 46, 108, 97, 110, 103, 46, 83, 116, 114, 105, 110, 103, 5, 119, 111, 114, 108, 100, 72, 16, 67, 111, 110, 115, 117, 109, 101, 114, 45, 65, 112, 112, 78, 97, 109, 101, 15, 104, 115, 102, 45, 115, 101, 114, 118, 101, 114, 45, 100, 101, 109, 111, 12, 116, 97, 114, 103, 101, 116, 95, 103, 114, 111, 117, 112, 4, 72, 83, 70, 49, 4, 95, 84, 73, 68, 78, 16, 101, 97, 103, 108, 101, 101, 121, 101, 95, 99, 111, 110, 116, 101, 120, 116, 72, 7, 116, 114, 97, 99, 101, 73, 100, 30, 48, 97, 57, 55, 54, 51, 54, 55, 49, 53, 51, 53, 48, 56, 50, 55, 56, 54, 49, 48, 55, 49, 48, 48, 49, 100, 48, 48, 54, 49, 5, 114, 112, 99, 73, 100, 1, 57, 16, 101, 97, 103, 108, 101, 69, 121, 101, 85, 115, 101, 114, 68, 97, 116, 97, 78, 90, 90, 12, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 11, 184, 12, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 11, 184, 12, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 11, 184}
	rpc := NewRPCHSF()
	reqs := rpc.SplitRequest(msg)
	reqsLen := len(reqs)
	if reqsLen != 1 {
		t.Errorf("%d != 1", reqsLen)
	} else {
		t.Log("split request follow with heart-beat msg ok")
	}
}

func Test_GetStreamId_01(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	rpc := NewRPCHSF()
	strId := rpc.GetStreamId(msg)
	if strId != "78" {
		t.Errorf("%s != 78", strId)
	} else {
		t.Log("get stream-id from request ok")
	}
}

func Test_GetStreamId_02(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 3, 'a', 'b', 'c'}
	rpc := NewRPCHSF()
	strId := rpc.GetStreamId(msg)
	if strId != "78" {
		t.Errorf("%s != 78", strId)
	} else {
		t.Log("get stream-id from response ok")
	}
}

func Test_GetStreamId_03(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 0, 2, 0, 0, 0, 0x01, 0x02}
	rpc := NewRPCHSF()
	strId := rpc.GetStreamId(msg)
	if strId != "" {
		t.Errorf("%s != ", strId)
	} else {
		t.Log("illegal length data ok")
	}
}

func Test_GetStreamId_04(t *testing.T) {
	msg := []byte{0x0e, 1, 2, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 3, 'a', 'b', 'c'}
	rpc := NewRPCHSF()
	strId := rpc.GetStreamId(msg)
	if strId != "" {
		t.Errorf("%s != ", strId)
	} else {
		t.Log("illegal type data ok")
	}
}

func Test_SetStreamId_01(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	rpc := NewRPCHSF()
	newId := "12345678"
	newMsg := rpc.SetStreamId(msg, newId)
	strId := rpc.GetStreamId(newMsg)
	if strId != newId {
		t.Errorf("%s != %s", strId, newId)
	} else {
		t.Log("set stream-id for request succ ok")
	}
}

func Test_SetStreamId_02(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 3, 'a', 'b', 'c'}
	rpc := NewRPCHSF()
	newId := "9876543"
	newMsg := rpc.SetStreamId(msg, newId)
	strId := rpc.GetStreamId(newMsg)
	if strId != newId {
		t.Errorf("%s(get) != %s(set)", strId, newId)
	} else {
		t.Log("set stream-id for response succ ok")
	}
}

func Test_SetStreamId_03(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 3, 'a', 'b', 'c'}
	rpc := NewRPCHSF()
	newId := "ok9876543"
	newMsg := rpc.SetStreamId(msg, newId)
	strId := rpc.GetStreamId(newMsg)
	if strId != "78" {
		t.Errorf("%s != 78", strId)
	} else {
		t.Log("set invalid stream-id ok")
	}
}

func Test_getHSFReqLen_01(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	len := getHSFReqLen(msg)
	if len != 37 {
		t.Errorf("%d != 37", len)
	} else {
		t.Log("getHSFReqLen succ ok")
	}
}

func Test_getHSFReqLen_02(t *testing.T) {
	msg := []byte{0x0f, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	len := getHSFReqLen(msg)
	if len != -1 {
		t.Errorf("%d != -1", len)
	} else {
		t.Log("getHSFReqLen illegal magic ok")
	}
}

func Test_getHSFReqLen_03(t *testing.T) {
	msg := []byte{0x0e, 2, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	len := getHSFReqLen(msg)
	if len != -1 {
		t.Errorf("%d != -1", len)
	} else {
		t.Log("getHSFReqLen unknown version ok")
	}
}

func Test_getHSFReqLen_04(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0}
	len := getHSFReqLen(msg)
	if len != -1 {
		t.Errorf("%d != -1", len)
	} else {
		t.Log("getHSFReqLen illegal length ok")
	}
}

func Test_getHSFReqLen_05(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 16, 0, 0, 0, 1, 0, 0, 0, 0, 'a', 'b', 'j', 'a', 'v', 'a', '.', 'l', 'a', 'n', 'g', '.', 'S', 't', 'r', 'i', 'n', 'g', 1}
	len := getHSFReqLen(msg)
	if len != 62 {
		t.Errorf("%d != 62", len)
	} else {
		t.Log("getHSFReqLen with argument succ ok")
	}
}

func Test_getHSFRspLen_01(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 3, 'a', 'b', 'c'}
	len := getHSFRspLen(msg)
	if len != 23 {
		t.Errorf("%d != 23", len)
	} else {
		t.Log("getHSFRspLen succ ok")
	}
}

func Test_getHSFRspLen_02(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78}
	len := getHSFRspLen(msg)
	if len != -1 {
		t.Errorf("%d != -1", len)
	} else {
		t.Log("getHSFRspLen illegal length ok")
	}
}

func Test_GetServiceName_01(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	rpc := NewRPCHSF()
	name := rpc.GetServiceName(msg)
	if name != "a" {
		t.Errorf("%s != a", name)
	} else {
		t.Log("GetServiceName succ ok")
	}
}

func Test_GetServiceName_02(t *testing.T) {
	msg := []byte{0x0e, 1, 1, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	rpc := NewRPCHSF()
	name := rpc.GetServiceName(msg)
	if name != "" {
		t.Errorf("%s != null", name)
	} else {
		t.Log("GetServiceName: is not request ok")
	}
}

func Test_GetServiceName_03(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'b'}
	rpc := NewRPCHSF()
	name := rpc.GetServiceName(msg)
	if name != "" {
		t.Errorf("%s != null", name)
	} else {
		t.Log("GetServiceName null service-name ok")
	}
}

func Test_GetMethodName_01(t *testing.T) {
	msg := []byte{0x0e, 1, 0, 2, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'a', 'b'}
	rpc := NewRPCHSF()
	name := rpc.GetMethodName(msg)
	if name != "b" {
		t.Errorf("%s != b", name)
	} else {
		t.Log("GetMethodName succ ok")
	}
}
