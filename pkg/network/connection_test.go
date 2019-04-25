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

package network

import (
	"fmt"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type MyEventListener struct{}

func (el *MyEventListener) OnEvent(event types.ConnectionEvent) {}

func testAddConnectionEventListener(n int, t *testing.T) {
	c := connection{}

	for i := 0; i < n; i++ {
		el0 := &MyEventListener{}
		c.AddConnectionEventListener(el0)
	}

	if len(c.connCallbacks) != n {
		t.Errorf("Expect %d, but got %d after AddConnectionEventListener(el0)", n, len(c.connCallbacks))
	}
}

func testAddBytesReadListener(n int, t *testing.T) {
	c := connection{}

	for i := 0; i < n; i++ {
		fn1 := func(bytesRead uint64) {}
		c.AddBytesReadListener(fn1)
	}

	if len(c.bytesReadCallbacks) != n {
		t.Errorf("Expect %d, but got %d after AddBytesReadListener(fn1)", n, len(c.bytesReadCallbacks))
	}
}

func testAddBytesSendListener(n int, t *testing.T) {
	c := connection{}

	for i := 0; i < n; i++ {
		fn1 := func(bytesSent uint64) {}
		c.AddBytesSentListener(fn1)
	}

	if len(c.bytesSendCallbacks) != n {
		t.Errorf("Expect %d, but got %d after AddBytesSentListener(fn1)", n, len(c.bytesSendCallbacks))
	}
}

func TestAddConnectionEventListener(t *testing.T) {
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("AddConnectionEventListener(%d)", i)
		t.Run(name, func(t *testing.T) {
			testAddConnectionEventListener(i, t)
		})
	}
}

func TestAddBytesReadListener(t *testing.T) {
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("AddBytesReadListener(%d)", i)
		t.Run(name, func(t *testing.T) {
			testAddBytesReadListener(i, t)
		})
	}
}

func TestAddBytesSendListener(t *testing.T) {
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("AddBytesSendListener(%d)", i)
		t.Run(name, func(t *testing.T) {
			testAddBytesSendListener(i, t)
		})
	}
}
