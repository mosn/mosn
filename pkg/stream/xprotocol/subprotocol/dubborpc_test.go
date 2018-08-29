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

func Test_dubbo_SplitFrame_01(t *testing.T) {
	msg := []byte{0xda, 0xbb, 2, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 3, '1', '2', '3'}
	rpc := NewRPCDubbo()
	reqs := rpc.SplitFrame(msg)
	reqsLen := len(reqs)
	if reqsLen != 1 {
		t.Errorf("%d != 1", reqsLen)
	} else {
		t.Log("split response succ ok")
	}
}

func Test_dubbo_SplitFrame_02(t *testing.T) {
	msg := []byte{0xda, 0xbb, 7, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 0}
	rpc := NewRPCDubbo()
	reqs := rpc.SplitFrame(msg)
	reqsLen := len(reqs)
	if reqsLen != 1 {
		t.Errorf("%d != 1", reqsLen)
	} else {
		t.Log("split heart-beat frame ok")
	}
}

func Test_dubbo_SplitFrame_03(t *testing.T) {
	msg := []byte{0xda, 0xbb, 3, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b', 0xda, 0xbb, 3, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 79, 0, 0, 0, 1, 'c', 0xda, 0xbb, 3, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 80, 0, 0, 0, 3, '1', '2', '3'}
	rpc := NewRPCDubbo()
	reqs := rpc.SplitFrame(msg)
	reqsLen := len(reqs)
	if reqsLen != 3 {
		t.Errorf("%d != 3", reqsLen)
	} else {
		t.Log("split mulit-request ok")
	}
}

func Test_dubbo_SplitFrame_04(t *testing.T) {
	msg := []byte{0xda, 0xbb, 3, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b', 0xda, 0xbb, 7, 0}
	rpc := NewRPCDubbo()
	reqs := rpc.SplitFrame(msg)
	reqsLen := len(reqs)
	if reqsLen != 1 {
		t.Errorf("%d != 1", reqsLen)
	} else {
		t.Log("split half-baked-request ok")
	}
}

func Test_dubbo_SplitFrame_05(t *testing.T) {
	msg := []byte{0xda, 0xbb, 3, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b', 0xda, 0xbb, 7, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 0, 0xda, 0xbb, 6, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 0}
	rpc := NewRPCDubbo()
	reqs := rpc.SplitFrame(msg)
	reqsLen := len(reqs)
	if reqsLen != 3 {
		t.Errorf("%d != 3", reqsLen)
	} else {
		t.Log("split request follow with heart-beat frame ok")
	}
}

func Test_dubbo_GetStreamID_01(t *testing.T) {
	msg := []byte{0xda, 0xbb, 3, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rpc := NewRPCDubbo()
	strId := rpc.GetStreamID(msg)
	if strId != "78" {
		t.Errorf("%s != 78", strId)
	} else {
		t.Log("get stream-id from request ok")
	}
}

func Test_dubbo_GetStreamID_02(t *testing.T) {
	msg := []byte{0xda, 0xbb, 2, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rpc := NewRPCDubbo()
	strId := rpc.GetStreamID(msg)
	if strId != "78" {
		t.Errorf("%s != 78", strId)
	} else {
		t.Log("get stream-id from response ok")
	}
}

func Test_dubbo_SetStreamID_01(t *testing.T) {
	msg := []byte{0xda, 0xbb, 3, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rpc := NewRPCDubbo()
	newId := "12345678"
	newMsg := rpc.SetStreamID(msg, newId)
	strId := rpc.GetStreamID(newMsg)
	if strId != newId {
		t.Errorf("%s != %s", strId, newId)
	} else {
		t.Log("set stream-id for request succ ok")
	}
}

func Test_dubbo_SetStreamID_02(t *testing.T) {
	msg := []byte{0xda, 0xbb, 2, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rpc := NewRPCDubbo()
	newId := "9876543"
	newMsg := rpc.SetStreamID(msg, newId)
	strId := rpc.GetStreamID(newMsg)
	if strId != newId {
		t.Errorf("%s(get) != %s(set)", strId, newId)
	} else {
		t.Log("set stream-id for response succ ok")
	}
}

func Test_dubbo_SetStreamID_03(t *testing.T) {
	msg := []byte{0xda, 0xbb, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rpc := NewRPCDubbo()
	newId := "ok9876543"
	newMsg := rpc.SetStreamID(msg, newId)
	strId := rpc.GetStreamID(newMsg)
	if strId != "78" {
		t.Errorf("%s != 78", strId)
	} else {
		t.Log("set invalid stream-id ok")
	}
}

func Test_dubbo_SetStreamID_04(t *testing.T) {
	msg := []byte{0xda, 0xbb, 4, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rpc := NewRPCDubbo()
	newId := "126"
	newMsg := rpc.SetStreamID(msg, newId)
	strId := rpc.GetStreamID(newMsg)
	if strId != newId {
		t.Errorf("%s != %s", strId, newId)
	} else {
		t.Log("set for heartbeat frame ok")
	}
}

func Test_isValidDubboData_01(t *testing.T) {
	msg := []byte{0xda, 0xbb, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rslt, len := isValidDubboData(msg)
	if rslt != true || len != 2 {
		t.Errorf("rslt(%v) != true, len=%d", rslt, len)
	} else {
		t.Log("isValidDubboData succ ok")
	}
}

func Test_isValidDubboData_02(t *testing.T) {
	msg := []byte{0xda, 0xb0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 2, 'a', 'b'}
	rslt, len := isValidDubboData(msg)
	if rslt != false || len != -1 {
		t.Errorf("rslt(%v) != false, len=%d", rslt, len)
	} else {
		t.Log("isValidDubboData illegal magic ok")
	}
}

func Test_isValidDubboData_03(t *testing.T) {
	msg := []byte{0xda, 0xbb, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78}
	rslt, len := isValidDubboData(msg)
	if rslt != false || len != -1 {
		t.Errorf("rslt(%v) != false, len=%d", rslt, len)
	} else {
		t.Log("isValidDubboData illegal header length ok")
	}
}

func Test_isValidDubboData_04(t *testing.T) {
	msg := []byte{0xda, 0xbb, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 78, 0, 0, 0, 3, 'a', 'b'}
	rslt, len := isValidDubboData(msg)
	if rslt != false || len != -1 {
		t.Errorf("rslt(%v) != false, len=%d", rslt, len)
	} else {
		t.Log("isValidDubboData illegal length ok")
	}
}
