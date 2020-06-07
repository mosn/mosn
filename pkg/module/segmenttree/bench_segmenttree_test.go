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

import "testing"

func BenchmarkTree_NewTree(b *testing.B) {
	nodeCount := 2000
	nodes := []Node{}
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, Node{
			Value:      i,
			RangeStart: uint64(i),
			RangeEnd:   uint64(i + 1),
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		NewTree(nodes, func(leftChildData, rightChildData interface{}) (currentNodeData interface{}) {
			return leftChildData.(int) + rightChildData.(int)
		})
	}
}

func BenchmarkTree_Update(b *testing.B) {
	nodeCount := 10000
	nodes := []Node{}
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, Node{
			Value:      i,
			RangeStart: uint64(i),
			RangeEnd:   uint64(i + 1),
		})
	}
	t := NewTree(nodes, func(leftChildData, rightChildData interface{}) (currentNodeData interface{}) {
		return leftChildData.(int) + rightChildData.(int)
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l, err := t.Leaf(100)
		if err != nil {
			b.Error(err)
		}
		l.Value = l.Value.(int) + 1
		t.Update(l)
	}

}
