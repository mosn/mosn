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
	"sync"
	"testing"
	"time"
)

type testData struct {
	data int
}

func assertTrue(t *testing.T, b bool) bool {
	if b {
		return true
	}
	t.Error("value is not true")
	return false
}
func waitGroupWithTimeout(t *testing.T, wg sync.WaitGroup) {
	// wait group with timeout
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()
	select {
	case <-c:
	case <-time.After(5 * time.Second):
		t.Error("case run timeout")
	}
}

func TestValue(t *testing.T) {
	data := &testData{data: 1}
	v := NewValue(data)
	if d, ok := v.Load().(*testData); assertTrue(t, ok) {
		if d.data != 1 {
			t.Errorf("data is not expected, expected : 1 , got: %d", d.data)
		}
		v.Put(d)
	}
	// test update
	if err := v.Update(&testData{data: 2}, 0); err != nil {
		t.Error("test update failed")
	}
	wg := sync.WaitGroup{}
	putback := make(chan struct{})
	if d, ok := v.Load().(*testData); assertTrue(t, ok) {
		if d.data != 2 {
			t.Error("data is not expected, expected: 2, got: ", d.data)
		}
		// delay put back
		wg.Add(1)
		go func() {
			<-putback
			v.Put(d)
			wg.Done()
		}()
	}
	start := make(chan struct{})
	finish := make(chan struct{})
	wg.Add(1)
	go func() {
		close(start)
		// test Update delayed return
		v.Update(&testData{data: 3}, 0)
		close(finish)
		wg.Done()
	}()
	<-start // wait goroutine run
	if err := v.Update(&testData{data: 4}, 0); err != Block {
		t.Error("expected update blocked by other update")
	}
	timer := time.NewTimer(5 * time.Second)
	wg.Add(1)
	once := sync.Once{}
Check:
	for {
		select {
		case <-finish:
			// update finish, new update can be run
			if err := v.Update(&testData{data: 4}, 0); err != nil {
				t.Error("expected update success")
			}
			break Check
		case <-timer.C:
			t.Error("case run timeout")
			break Check
		default: // not finish, but value is updated
			time.Sleep(10 * time.Millisecond) // wait Update sleep passed
			if d, ok := v.Load().(*testData); assertTrue(t, ok) {
				if d.data != 3 {
					t.Error("data is not expected, expected: 3, got: ", d.data)
				}
				v.Put(d)
			}
			// putback last Load, make update finish
			close(putback)
			once.Do(func() {
				wg.Done() // at least reach once
			})
		}
		time.Sleep(100 * time.Millisecond)
	}
	waitGroupWithTimeout(t, wg)
}

func TestUpdateTimeout(t *testing.T) {
	v := NewValue(&testData{data: 1})
	v.Load() //load a value, and never put back
	if err := v.Update(&testData{data: 2}, 500*time.Millisecond); err != Timeout {
		t.Error("expected update timeout")
	}
	// Update timeout, but value is updated
	if d, ok := v.Load().(*testData); assertTrue(t, ok) {
		if d.data != 2 {
			t.Error("data is not expected, expected: 2, got: ", d.data)
		}
	}
}

func TestUnexpectedPut(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer func() {
		waitGroupWithTimeout(t, wg)
	}()
	defer func() {
		// trigger panic
		if r := recover(); r != nil {
			wg.Done()
		}
	}()
	v := NewValue(nil)
	v.Put(nil) // put before load
}
