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

package rcu

import (
	"sync/atomic"
	"time"
	"unsafe"
)

// NewValue makes a value with data i
func NewValue(i interface{}) *Value {
	v := &Value{}
	v.Update(i, 0)
	return v
}

// Load returns the keeped data, data used count will add one
func (v *Value) Load() interface{} {
	i := v.element.Load()
	if i == nil {
		return nil
	}
	e := i.(*element)
	atomic.AddInt32(&e.count, 1)
	return e.i
}

// Put the data back, the used count will decrease one
// If put a data before Load, it will panic
func (v *Value) Put(i interface{}) {
	ptr := (*[2]unsafe.Pointer)(unsafe.Pointer(&i))[1]
	value := v.element.Load()
	if v == nil {
		return
	}
	e := value.(*element)
	if ptr == (*[2]unsafe.Pointer)(unsafe.Pointer(&e.i))[1] {
		if c := atomic.AddInt32(&e.count, -1); c < 0 {
			panic("put data before load")
		}
		return
	}
	value = v.expired.Load()
	if v == nil {
		return
	}
	e = value.(*element)
	if ptr == (*[2]unsafe.Pointer)(unsafe.Pointer(&e.i))[1] {
		if c := atomic.AddInt32(&e.count, -1); c < 0 {
			panic("put data before load")
		}
		return
	}
}

// Update can update the value directly, but will return success until the data used count is zero or reach timeout
// If it is reached timeout, it will returns a timeout error with value updated
// If a Update is not returned, the other Update will be blocked, and returns a block error without value updated
func (v *Value) Update(i interface{}, wait time.Duration) error {
	if !atomic.CompareAndSwapInt32(&v.running, 0, 1) {
		return Block
	}
	defer atomic.CompareAndSwapInt32(&v.running, 1, 0)
	e := &element{i: i}
	old := v.element.Load()
	if old != nil {
		v.expired.Store(old)
	}
	v.element.Store(e)

	if old == nil {
		return nil
	}
	e = old.(*element)

	interval := 10 * time.Millisecond
	// element load and element count add are two options, use sleep to wait
	time.Sleep(interval)

	if wait <= 0 {
		wait = 5 * time.Second
	}
	stop := time.NewTimer(wait)
	for atomic.LoadInt32(&e.count) != 0 {
		select {
		case <-stop.C:
			return Timeout
		default:
			time.Sleep(interval)
		}
	}
	return nil
}
