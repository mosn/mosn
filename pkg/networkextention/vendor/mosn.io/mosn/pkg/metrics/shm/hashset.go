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

package shm

import (
	"errors"
	"reflect"
	"strconv"
	"unsafe"
)

var (
	hashEntrySize = int(unsafe.Sizeof(hashEntry{}))
	hashMetaSize  = int(unsafe.Sizeof(meta{}))
)

// hash
const (
	// offset32 FNVa offset basis. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	offset32 = 2166136261
	// prime32 FNVa prime value. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	prime32 = 16777619

	// indicate end of linked-list
	sentinel = 0xffffffff
)

// gets the string and returns its uint32 hash value.
func hash(key string) uint32 {
	var hash uint32 = offset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= prime32
	}

	return hash
}

type hashSet struct {
	entry []hashEntry
	meta  *meta
	slots []uint32
}

type hashEntry struct {
	metricsEntry
	next uint32

	// Prevents false sharing on widespread platforms with
	// 128 mod (cache line size) = 0 .
	pad [128 - unsafe.Sizeof(metricsEntry{})%128 - 4]byte
}

type meta struct {
	cap       uint32
	size      uint32
	freeIndex uint32

	slotsNum uint32
	bytesNum uint32
}

func newHashSet(segment uintptr, bytesNum, cap, slotsNum int, init bool) (*hashSet, error) {
	set := &hashSet{}

	// 1. entry mapping
	entrySlice := (*reflect.SliceHeader)(unsafe.Pointer(&set.entry))
	entrySlice.Data = segment
	entrySlice.Len = cap
	entrySlice.Cap = cap

	offset := cap * hashEntrySize
	if offset > bytesNum {
		return nil, errors.New("segment is not enough to map hashSet.entry")
	}

	// 2. meta mapping
	set.meta = (*meta)(unsafe.Pointer(segment + uintptr(offset)))
	set.meta.slotsNum = uint32(slotsNum)
	set.meta.bytesNum = uint32(bytesNum)
	set.meta.cap = uint32(cap)

	offset += hashMetaSize
	if offset > bytesNum {
		return nil, errors.New("segment is not enough to map hashSet.meta")
	}

	// 3. slots mapping
	slotSlice := (*reflect.SliceHeader)(unsafe.Pointer(&set.slots))
	slotSlice.Data = segment + uintptr(offset)
	slotSlice.Len = slotsNum
	slotSlice.Cap = slotsNum

	offset += 4 * slotsNum // slot type is uint32
	if offset > bytesNum {
		return nil, errors.New("segment is not enough to map hashSet.slots")
	}

	if init {
		// 4. initialize
		// 4.1 meta
		set.meta.size = 0
		set.meta.freeIndex = 0

		// 4.2 slots
		for i := 0; i < slotsNum; i++ {
			set.slots[i] = sentinel
		}

		// 4.3 entries
		last := cap - 1
		for i := 0; i < last; i++ {
			set.entry[i].next = uint32(i + 1)
		}
		set.entry[last].next = sentinel
	}
	return set, nil
}

func (s *hashSet) Alloc(name string) (*hashEntry, bool) {
	// 1. search existed slots and entries
	h := hash(name)
	slot := h % s.meta.slotsNum

	// name convert if length exceeded
	if len(name) > maxNameLength {
		// if name is longer than max length, use hash_string as leading character
		// and the remaining maxNameLength - len(hash_string) bytes follows
		hStr := strconv.Itoa(int(h))
		name = hStr + name[len(hStr)+len(name)-maxNameLength:]
	}

	nameBytes := []byte(name)

	var entry *hashEntry
	for index := s.slots[slot]; index != sentinel; {
		entry = &s.entry[index]

		if entry.equalName(nameBytes) {
			return entry, false
		}

		index = entry.next
	}

	// 2. create new entry
	if s.meta.size >= s.meta.cap {
		return nil, false
	}

	newIndex := s.meta.freeIndex
	newEntry := &s.entry[newIndex]
	newEntry.assignName(nameBytes)
	newEntry.ref = 1

	if entry == nil {
		s.slots[slot] = newIndex
	} else {
		entry.next = newIndex
	}

	s.meta.size++
	s.meta.freeIndex = newEntry.next
	newEntry.next = sentinel

	return newEntry, true
}

func (s *hashSet) Free(entry *hashEntry) {
	if entry.decRef() {
		name := string(entry.getName())

		// 1. search existed slots and entries
		h := hash(name)
		slot := h % s.meta.slotsNum

		var index uint32
		var prev *hashEntry
		for index = s.slots[slot]; index != sentinel; {
			target := &s.entry[index]
			if entry == target {
				break
			}

			prev = target
			index = target.next
		}

		// 2. unlink, re-init and add to the head of free list
		if prev != nil {
			prev.next = entry.next
		} else {
			s.slots[slot] = entry.next
		}

		*entry = hashEntry{}

		entry.next = s.meta.freeIndex
		s.meta.freeIndex = index
	}
}
