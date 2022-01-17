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
	"sync"
	"time"
)

type edfSchduler struct {
	lock        sync.Mutex
	items       *edfHeap
	currentTime float64
}

func newEdfScheduler(cap int) *edfSchduler {
	return &edfSchduler{
		items: newEdfHeap(cap),
	}
}

// edfEntry is an internal wrapper for item that also stores weight and relative position in the queue.
type edfEntry struct {
	deadline   float64
	weight     float64
	item       WeightItem
	queuedTime time.Time
}

type WeightItem interface {
	Weight() uint32
}

// Add new item into the edfSchduler
func (edf *edfSchduler) Add(item WeightItem, weight float64) {
	edf.lock.Lock()
	defer edf.lock.Unlock()
	entry := edfEntry{
		deadline:   edf.currentTime + 1.0/weight,
		weight:     weight,
		item:       item,
		queuedTime: time.Now(),
	}
	edf.items.Push(&entry)
}

// Pick entry with closest deadline and push again
// Note that you need to ensure the return result of weightFunc is not equal to 0
func (edf *edfSchduler) NextAndPush(weightFunc func(item WeightItem) float64) interface{} {
	edf.lock.Lock()
	defer edf.lock.Unlock()
	if edf.items.Empty() {
		return nil
	}
	entry := edf.items.Peek()
	edf.currentTime = entry.deadline
	weight := weightFunc(entry.item)
	// update the index、deadline and put into priorityQueue again
	entry.deadline = entry.deadline + 1.0/weight
	entry.weight = weight
	entry.queuedTime = time.Now()
	edf.items.Fix(0)
	return entry.item
}
