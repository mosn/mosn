package cluster

import (
	"container/heap"
	"sync"
)

type edfSchduler struct {
	lock  sync.Mutex
	items PriorityQueue
}

func newEdfScheduler() *edfSchduler {
	return &edfSchduler{}
}

// edfEntry is an internal wrapper for item that also stores weight and relative position in the queue.
type edfEntry struct {
	deadline float64
	weight   uint32
	item     interface{}
}

// PriorityQueue
type PriorityQueue []*edfEntry

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].deadline < pq[j].deadline }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*edfEntry))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	*pq = old[0 : len(old)-1]
	return old[len(old)-1]
}


// Current time in EDF scheduler.
func (edf *edfSchduler) currentTime() float64 {
	if len(edf.items) == 0 {
		return 0.0
	}
	return edf.items[0].deadline
}

func (edf *edfSchduler) Add(item interface{}, weight uint32) {
	edf.lock.Lock()
	defer edf.lock.Unlock()
	entry := edfEntry{
		deadline: edf.currentTime() + 1.0/float64(weight),
		weight:   weight,
		item:     item,
	}
	heap.Push(&edf.items, &entry)
}

func (edf *edfSchduler) Next() interface{} {
	edf.lock.Lock()
	defer edf.lock.Unlock()
	if len(edf.items) == 0 {
		return nil
	}
	item := edf.items[0]
	item.deadline = edf.currentTime() + 1.0/float64(item.weight)
	heap.Fix(&edf.items, 0)
	return item.item
}
