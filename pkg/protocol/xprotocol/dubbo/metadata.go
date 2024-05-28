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

package dubbo

import "sync"

// DubboPubMetadata dubbo pub cache metadata
var DubboPubMetadata = &Metadata{}

// DubboSubMetadata dubbo sub cache metadata
var DubboSubMetadata = &Metadata{}

// Metadata cache service pub or sub metadata.
// speed up for decode or encode dubbo performance.
// please do not use outside of the dubbo framework.
type Metadata struct {
	data map[string]*Node
	mu   sync.RWMutex // protect data internal
}

// Find cached pub or sub metadata.
// caller should check match is true
func (m *Metadata) Find(path, version string) (node *Node, matched bool) {
	// we found nothing
	if m.data == nil {
		return nil, false
	}

	m.mu.RLocker().Lock()
	// for performance
	// m.mu.RLocker().Unlock() should be called.

	// we check head node first
	head := m.data[path]
	if head == nil || head.count <= 0 {
		m.mu.RLocker().Unlock()
		return nil, false
	}

	node = head.Next
	// just only once, just return
	// for dubbo framework, that's what we're expected.
	if head.count == 1 {
		m.mu.RLocker().Unlock()
		return node, true
	}

	var count int
	var found *Node

	for ; node != nil; node = node.Next {
		if node.Version == version {
			if found == nil {
				found = node
			}
			count++
		}
	}

	m.mu.RLocker().Unlock()
	return found, count == 1
}

// Register pub or sub metadata
func (m *Metadata) Register(path string, node *Node) {
	m.mu.Lock()
	// for performance
	// m.mu.Unlock() should be called.

	if m.data == nil {
		m.data = make(map[string]*Node, 4)
	}

	// we check head node first
	head := m.data[path]
	if head == nil {
		head = &Node{
			count: 1,
		}
		// update head
		m.data[path] = head
	}

	insert := &Node{
		Service: node.Service,
		Version: node.Version,
		Group:   node.Group,
	}

	next := head.Next
	if next == nil {
		// fist insert, just insert to head
		head.Next = insert
		// record last element
		head.last = insert
		m.mu.Unlock()
		return
	}

	// we check already exist first
	for ; next != nil; next = next.Next {
		// we found it
		if next.Version == node.Version && next.Group == node.Group {
			// release lock and no nothing
			m.mu.Unlock()
			return
		}
	}

	head.count++
	// append node to the end of the list
	head.last.Next = insert
	// update last element
	head.last = insert
	m.mu.Unlock()
}

// Contains check if cached pub or sub metadata strict exists.
// caller should check match is true
func (m *Metadata) Contains(path, version, group string) (node *Node, matched bool) {
	// we found nothing
	if m.data == nil {
		return nil, false
	}

	m.mu.RLocker().Lock()
	// for performance
	// m.mu.RLocker().Unlock() should be called.

	// we check head node first
	head := m.data[path]
	if head == nil || head.count <= 0 {
		m.mu.RLocker().Unlock()
		return nil, false
	}

	node = head.Next
	// just only once, just return
	// for dubbo framework, that's what we're expected.
	if head.count == 1 {
		m.mu.RLocker().Unlock()
		return node, node.Version == version && node.Group == group
	}

	var count int
	var found *Node

	for ; node != nil; node = node.Next {
		if node.Version == version && node.Group == group {
			if found == nil {
				found = node
			}
			count++
		}
	}

	m.mu.RLocker().Unlock()
	return found, count == 1
}

// Clear all cache data
func (m *Metadata) Clear() {
	m.mu.Lock()
	// help for gc
	m.data = nil
	m.mu.Unlock()
}

// Node pub or sub host metadata,
// Keep the structure as simple as possible.
type Node struct {
	Service string // interface name
	Version string // interface version
	Group   string // interface group
	count   int    // number of node elements
	Next    *Node  // next node
	last    *Node  // last node
}
