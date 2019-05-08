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

package store

import (
	"sync/atomic"
	"testing"
)

func TestOnStateChanged(t *testing.T) {
	calls := map[State]uint32{
		Init:                  0,
		Running:               0,
		Active_Reconfiguring:  0,
		Passive_Reconfiguring: 0,
	}
	onChanged := func(s State) {
		cnt, ok := calls[s]
		if ok {
			calls[s] = atomic.AddUint32(&cnt, 1)
		}
	}
	RegisterOnStateChanged(onChanged)
	for _, s := range []State{
		Init, Running, Active_Reconfiguring, Passive_Reconfiguring,
	} {
		for i := 0; i < 10; i++ {
			SetMosnState(s)
		}
	}
	for s, cnt := range calls {
		if cnt != 10 {
			t.Errorf("state:%d count is %d", s, cnt)
		}
	}
}
