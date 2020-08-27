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

package protocol

import (
	"context"
	"testing"
)

func TestBuffer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestBuffer error: %v", r)
		}
	}()

	buf := ProtocolBuffersByContext(context.Background())
	if buf == nil {
		t.Error("New buffer failed.")
	}

	if d := buf.GetReqData(1024); d == nil {
		t.Error("GetReqData failed.")
	}

	if d := buf.GetReqHeader(1024); d == nil {
		t.Error("GetReqHeader failed.")
	}

	if d := buf.GetReqTailers(); d == nil {
		t.Error("GetReqTailers failed.")
	}

	if d := buf.GetRspData(1024); d == nil {
		t.Error("GetRspData failed.")
	}

	if d := buf.GetRspHeader(1024); d == nil {
		t.Error("GetRspHeader failed.")
	}

	if d := buf.GetRspTailers(); d == nil {
		t.Error("GetRspTailers failed.")
	}

	p := protocolBufferCtx{}
	p.Reset(buf)

}
