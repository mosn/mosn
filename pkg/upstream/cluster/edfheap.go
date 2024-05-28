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

const minCap = 16

type LessFunc func(a, b *edfEntry) bool

// edfHeap type-specific heap for edfEntry
type edfHeap struct {
	elements []*edfEntry
	size     int
}

func edfEntryLess(a, b *edfEntry) bool {
	if a.deadline == b.deadline {
		return a.queuedTime < b.queuedTime
	}
	return a.deadline < b.deadline
}

func newEdfHeap(cap int) *edfHeap {
	if cap < minCap {
		cap = minCap
	}
	return &edfHeap{
		elements: make([]*edfEntry, cap),
	}
}

func (h *edfHeap) Fix(i int) {
	if !h.fixDown(i, h.size) {
		h.fixUp(i)
	}
}

func (h *edfHeap) Push(element *edfEntry) {
	n := h.size
	h.size++
	h.elements[n] = element
	h.fixUp(n)
}

func (h *edfHeap) Peek() *edfEntry {
	return h.elements[0]
}

func (h *edfHeap) Pop() *edfEntry {
	value := h.elements[0]
	h.size--
	n := h.size
	h.elements[0] = h.elements[n]
	h.elements[n] = nil // For gc
	h.fixDown(0, n)
	return value
}

func (h *edfHeap) Size() int {
	return h.size
}

func (h *edfHeap) Empty() bool {
	return h.size == 0
}

func (h *edfHeap) fixUp(i int) bool {
	oldI := i
	var parent int
	var element *edfEntry
	// Find the largest element position from the current node to the root
	// that is smaller than the current value, and move all elements on the path down
	for element = h.elements[i]; i > 0; i = parent {
		parent = (i - 1) / 2
		if edfEntryLess(element, h.elements[parent]) {
			h.elements[i] = h.elements[parent]
		} else {
			break
		}
	}
	if oldI == i {
		return false
	}
	h.elements[i] = element
	return true
}

func (h *edfHeap) fixDown(i, n int) bool {
	oldI := i
	var child int
	var element *edfEntry
	for element = h.elements[i]; ; i = child {
		child = i*2 + 1
		// In order to reduce one calculation
		if child < n {
			// According to the heap order, select the element with the smallest value
			// in the left and right child nodes as the value of the parent node
			if child+1 < n && edfEntryLess(h.elements[child+1], h.elements[child]) {
				child++
			}
			if edfEntryLess(h.elements[child], element) {
				h.elements[i] = h.elements[child]
			} else {
				break
			}
		} else {
			break
		}
	}
	if oldI == i {
		return false
	}
	h.elements[i] = element
	return true
}
