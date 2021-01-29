// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import "sync/atomic"

type AtomicBool struct {
	// default 0, means false
	flag int32
}

func (b *AtomicBool) CompareAndSet(old, new bool) bool {
	if old == new {
		return true
	}
	var oldInt, newInt int32
	if old {
		oldInt = 1
	}
	if new {
		newInt = 1
	}
	return atomic.CompareAndSwapInt32(&(b.flag), oldInt, newInt)
}

func (b *AtomicBool) Set(value bool) {
	i := int32(0)
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), int32(i))
}

func (b *AtomicBool) Get() bool {
	if atomic.LoadInt32(&(b.flag)) != 0 {
		return true
	}
	return false
}
