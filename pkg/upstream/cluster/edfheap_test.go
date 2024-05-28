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

package cluster

import (
	"math"
	"math/rand"
	"testing"
)

func (h *edfHeap) verify(t *testing.T, i int) {
	t.Helper()
	n := h.size
	left := 2*i + 1
	right := 2*i + 2
	if left < n {
		if edfEntryLess(h.elements[left], h.elements[i]) {
			t.Errorf("heap invariant invalidated [%d] = %v > [%d] = %v", i, h.elements[i], left, h.elements[left])
		}
		h.verify(t, left)
	}
	if right < n {
		if edfEntryLess(h.elements[left], h.elements[i]) {
			t.Errorf("heap invariant invalidated [%d] = %v > [%d] = %v", i, h.elements[i], right, h.elements[right])
		}
		h.verify(t, right)
	}
}

func TestEdfHeap_PushAndPop(t *testing.T) {
	h := newEdfHeap(100)
	// Test that the push operation conforms to heap ordering
	for i := 0; i < 50; i++ {
		h.Push(&edfEntry{
			deadline: float64(i / 4), // for same deadline
			weight:   rand.Float64(),
		})
	}
	h.verify(t, 0)

	for i := 50; i < 100; i++ {
		h.Push(&edfEntry{
			deadline: float64(i / 4), // for same deadline
			weight:   rand.Float64(),
		})
		h.verify(t, 0)
	}
	if h.Size() != 100 {
		t.Errorf("Expected heap size to be 100, but was %d", h.Size())
	}

	// Test that the return result of the pop operation is non-decreasing
	var preValue *edfEntry = nil
	for !h.Empty() {
		value := h.Pop()
		if preValue == nil {
			preValue = value
		} else {
			if edfEntryLess(value, preValue) {
				t.Errorf("Heap pop does not conform to ordering, preValue = %v, value = %v", preValue, value)
			}
		}
	}

	if h.Size() != 0 {
		t.Errorf("Expected heap size to be 0, but was %d", h.Size())
	}
}

func TestEdfHeap_PushAndPopNotChangeCapacity(t *testing.T) {
	h := newEdfHeap(100)
	for i := 0; i < 100; i++ {
		h.Push(&edfEntry{
			deadline: float64(i / 4), // for same deadline
			weight:   rand.Float64(),
		})
	}
	for !h.Empty() {
		h.Pop()
	}
	if cap(h.elements) != 100 {
		t.Errorf("Heap continuous push and continuous pop should not change the capacity. cap = %d", cap(h.elements))
	}
}

func TestEdfHeap_Fix(t *testing.T) {
	h := newEdfHeap(100)
	for i := 0; i < 100; i++ {
		h.Push(&edfEntry{
			deadline: float64(i / 4),
			weight:   rand.Float64(),
		})
	}
	h.verify(t, 0)

	for i := 0; i < 100; i++ {
		e := h.Peek()
		e.deadline += 1 / e.weight
		h.Fix(0)
		h.verify(t, 0)
	}
}

func TestEdfHeap_edfEntryLess(t *testing.T) {
	tests := []struct {
		a    *edfEntry
		b    *edfEntry
		less bool
	}{
		{&edfEntry{deadline: 1.0, queuedTime: 1}, &edfEntry{deadline: 1.0, queuedTime: 2}, true},
		{&edfEntry{deadline: 1.0, queuedTime: 2}, &edfEntry{deadline: 1.0, queuedTime: 1}, false},
		{&edfEntry{deadline: 1.0, queuedTime: 1}, &edfEntry{deadline: 1.0, queuedTime: 1}, false},
		{&edfEntry{deadline: 2.0, queuedTime: 1}, &edfEntry{deadline: 1.0, queuedTime: 1}, false},
		{&edfEntry{deadline: 1.0, queuedTime: 1}, &edfEntry{deadline: 2.0, queuedTime: 1}, true},
		{&edfEntry{deadline: 1.0, queuedTime: 2}, &edfEntry{deadline: 2.0, queuedTime: 1}, true},
	}

	for _, tst := range tests {
		if edfEntryLess(tst.a, tst.b) != tst.less {
			t.Fatalf("%v < %v must be %v", tst.a, tst.b, tst.less)
		}
	}
}

func BenchmarkEdfHeap_Push(b *testing.B) {
	b.StopTimer()
	h := newEdfHeap(b.N)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		h.Push(&edfEntry{weight: rand.Float64()})
	}
}

func BenchmarkEdfHeap_Pop(b *testing.B) {
	b.StopTimer()
	h := newEdfHeap(b.N)
	for i := 0; i < b.N; i++ {
		h.Push(&edfEntry{weight: rand.Float64()})
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		h.Pop()
	}
}

func BenchmarkEdfHeap_Fix(b *testing.B) {
	b.StopTimer()
	h := newEdfHeap(b.N)
	for i := 0; i < b.N; i++ {
		h.Push(&edfEntry{weight: rand.Float64()})
	}
	b.StartTimer()
	// Max tree height
	logN := math.Log2(float64(b.N))
	for i := 0; i < b.N; i++ {
		e := h.Peek()
		e.weight += logN
		h.Fix(0)
	}
}
