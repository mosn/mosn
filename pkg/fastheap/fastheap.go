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

package fastheap

const minCap = 16

type LessFunc func(a, b interface{}) bool

// Heap a faster implementation than golang's heap
type Heap struct {
	elements []interface{}
	size     int
	less     LessFunc
}

func New(less LessFunc) *Heap {
	return NewWithCap(minCap, less)
}

func NewWithCap(cap int, less LessFunc) *Heap {
	if cap < minCap {
		cap = minCap
	}
	return &Heap{
		elements: make([]interface{}, cap),
		less:     less,
	}
}

func (h *Heap) Fix(i int) {
	if !h.fixDown(i, h.size) {
		h.fixUp(i)
	}
}

func (h *Heap) Push(element interface{}) {
	h.ensureIncrement()
	n := h.size
	h.size++
	h.elements[n] = element
	h.fixUp(n)
}

func (h *Heap) Peek() interface{} {
	return h.elements[0]
}

func (h *Heap) Pop() interface{} {
	value := h.elements[0]
	h.size--
	n := h.size
	h.elements[0] = h.elements[n]
	h.elements[n] = nil // For gc
	h.fixDown(0, n)
	h.ensureDecrement() // Avoid only growing but not decreasing
	return value
}

func (h *Heap) Size() int {
	return h.size
}

func (h *Heap) Empty() bool {
	return h.size == 0
}

func (h *Heap) ensureIncrement() {
	if h.size+1 > cap(h.elements) {
		oldElements := h.elements
		h.elements = make([]interface{}, 2*cap(h.elements))
		copy(h.elements, oldElements)
	}
}

func (h *Heap) ensureDecrement() {
	if h.size*2 < cap(h.elements) {
		newCap := cap(h.elements) / 2
		if newCap < minCap {
			newCap = minCap
		}
		oldElements := h.elements
		h.elements = make([]interface{}, newCap)
		copy(h.elements, oldElements)
	}
}

func (h *Heap) fixUp(i int) bool {
	oldI := i
	var parent int
	var element interface{}
	for element = h.elements[i]; i > 0; i = parent {
		parent = (i - 1) / 2
		if h.less(element, h.elements[parent]) {
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

func (h *Heap) fixDown(i, n int) bool {
	oldI := i
	var child int
	var element interface{}
	for element = h.elements[i]; ; i = child {
		child = i*2 + 1
		// In order to reduce one calculation
		if child < n {
			// According to the heap order, select the element with the smallest value
			// in the left and right child nodes as the value of the parent node
			if child+1 < n && h.less(h.elements[child+1], h.elements[child]) {
				child++
			}
			if h.less(h.elements[child], element) {
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

// AssertHeapOrdering for testing
func (h *Heap) AssertHeapOrdering(onFail func(i int, element interface{})) {
	for i := 0; i < h.size; i++ {
		if (i*2+1 < h.size && h.less(h.elements[i*2+1], h.elements[i])) ||
			(i*2+2 < h.size && h.less(h.elements[i*2+2], h.elements[i])) {
			onFail(i, h.elements[i])
		}
	}
}
