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
	"testing"
)

func TestCommonHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestCommonHeader error: %v", r)
		}
	}()

	header := CommonHeader{MosnHeaderHostKey: "test"}
	if v, ok := header.Get(MosnHeaderHostKey); !ok || v != "test" {
		t.Error("Get header failed.")
	}

	header.Set(MosnHeaderHostKey, "test1")
	if v, ok := header.Get(MosnHeaderHostKey); !ok || v != "test1" {
		t.Error("Set header failed.")
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

	if l := h2.ByteSize(); l != uint64(len(MosnHeaderHostKey)+len("test")) {
		t.Errorf("get ByteSize failed got: %d want:%d", l, len(MosnHeaderHostKey)+len("test"))
	}

}
