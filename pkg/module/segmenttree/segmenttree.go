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

package segmenttree

import (
	"fmt"
	"sync"
)

// SegmentTreeUpdateFunc receive data of left-child node and right-child node
// calculate and return the current node data
// it can be implement by user and pass as param when creating segment tree
type SegmentTreeUpdateFunc func(leftChildData, rightChildData interface{}) (currentNodeData interface{})

// Tree is a segment tree structure.
// Tree use data field to save node data, which is a interface list,
// and rangeStart & rangeEnd map to save node segment.
// Therefore, data in param 'nodes' of NewTree method below, will be fill in these fields.
//
// Data field's size is twice of leaf count, leaf node data and non-leaf node data in data field is like:
//  content         [----non-leaf node data----][--------leaf node data-------]
// data list index  |0--------------leafCount-1||leafCount-------2*leafCount-1]
//
// For example, use 4 nodes with data 1, 2, 3, 4 to create a tree, and a updateFunc of adding sub-tree data,
// and data layout will be like:
//     content     [nil,10(3+7),3(1+2),7(3+4)][1,2,3,4]
// data list index [0,   1,     2,          3][4,5,6,7]
//
// Why this data field layout? Tree's update func will first put list data at the second half of data field,
// and then calculate the first half of data field, by adding two data at a time reversely, and put it in the
// first half of data field, back to front.
type Tree struct {
	// data save leaf node data and non-leaf node data
	// data size = 2 * leafCount
	data []interface{}
	// rangeStart and rangeEnd saves segments.
	// Map index is the leaf node / non-leaf node index
	// and map value is segment start / end.
	rangeStart map[int]uint64
	rangeEnd   map[int]uint64
	leafCount  int
	updateFunc SegmentTreeUpdateFunc
	// mutex lock for update or retrieve tree non-leaf node data
	mutex sync.Mutex
}

// Leaf returns the leaf node
// Because leaf node's data will not change, no need th lock tree's mutex.
func (t *Tree) Leaf(index int) (*Node, error) {
	if index >= t.leafCount {
		return nil, fmt.Errorf("index %d out of range: %d", index, t.leafCount)
	}

	leafIndex := t.leafCount + index
	data := t.data[leafIndex]
	rangeStart := t.rangeStart[leafIndex]
	rangeEnd := t.rangeEnd[leafIndex]

	return &Node{
		Value:      data,
		index:      leafIndex,
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
	}, nil
}

// Update segment tree data
// by saving leaf node data and calling updateFunc to update parent non-leaf node data.
func (t *Tree) Update(n *Node) {
	// TODO check if node is a leaf node
	t.mutex.Lock()
	defer t.mutex.Unlock()
	index := n.index
	// update current node data
	t.data[index] = n.Value

	// find root index
	leftIndex := index
	rightIndex := index + 1
	if index%2 == 1 {
		// if index is odd, means that it's a right child
		rightIndex = index
		leftIndex = index - 1
	}
	parentIndex := leftIndex / 2

	for parentIndex > 0 {
		t.data[parentIndex] = t.updateFunc(t.data[leftIndex], t.data[rightIndex])

		leftIndex = parentIndex
		rightIndex = leftIndex + 1
		if parentIndex%2 == 1 {
			rightIndex = parentIndex
			leftIndex = parentIndex - 1
		}
		parentIndex /= 2
	}
}

// Node struct save value and segment of a segment tree.
// A node structure can be a leaf node, or a non-leaf node.
//
// Index field in Node structure saves data field index of a segment tree.
type Node struct {
	Value interface{}
	// segment tree's data field index
	index      int
	RangeStart uint64
	RangeEnd   uint64
}

func NewTree(nodes []Node, updateFunc SegmentTreeUpdateFunc) *Tree {
	t := &Tree{
		updateFunc: updateFunc,
		leafCount:  len(nodes),
		mutex:      sync.Mutex{},
	}
	t.data, t.rangeStart, t.rangeEnd = build(nodes, updateFunc)

	return t
}

// build use nodes list and updateFunc to produce data list, rangeStart & rangeEnd map
func build(nodes []Node, updateFunc SegmentTreeUpdateFunc) (data []interface{}, rangeStart, rangeEnd map[int]uint64) {
	if len(nodes) == 0 {
		return nil, nil, nil
	}
	count := len(nodes)

	data = make([]interface{}, 2*count)
	rangeStart = make(map[int]uint64)
	rangeEnd = make(map[int]uint64)

	for i := 0; i < count; i++ {
		data[count+i] = nodes[i].Value
		rangeStart[count+i] = nodes[i].RangeStart
		rangeEnd[count+i] = nodes[i].RangeEnd
	}

	n := 2*count - 1
	for {
		leftIndex := n - 1
		rightIndex := n
		rootIndex := leftIndex / 2

		data[rootIndex] = updateFunc(data[leftIndex], data[rightIndex])
		rangeStart[rootIndex] = rangeStart[leftIndex]
		if rangeStart[rightIndex] < rangeStart[leftIndex] {
			rangeStart[rootIndex] = rangeStart[rightIndex]
		}

		rangeEnd[rootIndex] = rangeEnd[leftIndex]
		if rangeEnd[rightIndex] > rangeEnd[leftIndex] {
			rangeEnd[rootIndex] = rangeEnd[rightIndex]
		}

		n -= 2

		if n/2 == 0 {
			break
		}
	}

	return
}

// FindParent return the parent node of current node.
// Root node return nil pointer for root has no parent.
// Because non-leaf node's data may be changed, need to acquire tree's mutex
func (t *Tree) FindParent(currentNode *Node) *Node {
	if currentNode.IsRoot() {
		return nil
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()
	rootIndex := currentNode.index / 2
	root := &Node{
		Value:      t.data[rootIndex],
		index:      rootIndex,
		RangeStart: t.rangeStart[rootIndex],
		RangeEnd:   t.rangeEnd[rootIndex],
	}
	return root
}

func (n *Node) IsRoot() bool {
	// root node's index is 1
	return n.index/2 == 0
}
