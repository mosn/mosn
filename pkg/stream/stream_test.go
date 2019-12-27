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

package stream

import (
	"errors"
	"testing"

	"mosn.io/mosn/pkg/types"
)

type event struct {
}

func (e *event) OnResetStream(reason types.StreamResetReason) {
	if e == nil {
		panic(errors.New("EventListener is nil"))
	}
}

func (e *event) OnDestroyStream() {
	if e == nil {
		panic(errors.New("EventListener is nil"))
	}
}

func TestStreamEventListener(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TestStreamEventListener error: %v", r)
		}
	}()

	var base BaseStream

	for i := 0; i < 10; i++ {
		go func() {
			for i := 0; i < 1000; i++ {
				base.AddEventListener(&event{})
			}
		}()
	}

	for i := 0; i < 100000; i++ {
		base.ResetStream(types.StreamLocalReset)
	}
}
