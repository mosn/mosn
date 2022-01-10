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

func TestHeap_PushAndPop(t *testing.T) {
	h := newEdfHeap(16)
	// Test that the push operation conforms to heap ordering
	for i := 0; i < 100; i++ {
		h.Push(&edfEntry{
			weight: rand.Float64(),
		})
	}
	h.assertHeapOrdering(func(i int, element interface{}) {
		t.Errorf("Element[%d] = %v does not conform to heap ordering", i, element)
	})

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

	// Test the minimum capacity after pop operation
	if cap(h.elements) != minCap {
		t.Errorf("Heap elements cap after decrement should not be less than %d", minCap)
	}
}

func BenchmarkHeap_Push(b *testing.B) {
	b.StopTimer()
	h := newEdfHeap(16)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		h.Push(&edfEntry{weight: rand.Float64()})
	}
}

func BenchmarkHeap_Pop(b *testing.B) {
	b.StopTimer()
	h := newEdfHeap(16)
	for i := 0; i < b.N; i++ {
		h.Push(&edfEntry{weight: rand.Float64()})
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		h.Pop()
	}
}

func BenchmarkHeap_Fix(b *testing.B) {
	b.StopTimer()
	h := newEdfHeap(16)
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
