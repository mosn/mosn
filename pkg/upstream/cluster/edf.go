package cluster

import (
	"container/heap"
	"sync"
	"time"
)

type edfSchduler struct {
	lock        sync.Mutex
	items       PriorityQueue
	currentTime float64
}

func newEdfScheduler(cap int) *edfSchduler {
	return &edfSchduler{
		items: make(PriorityQueue, 0, cap),
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

// PriorityQueue
type PriorityQueue []*edfEntry

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
	heap.Push(&edf.items, &entry)
}

// Pick entry with closest deadline and push again
// Note that you need to ensure the return result of weightFunc is not equal to 0
func (edf *edfSchduler) NextAndPush(weightFunc func(item WeightItem) float64) interface{} {
	edf.lock.Lock()
	defer edf.lock.Unlock()
	if len(edf.items) == 0 {
		return nil
	}
	entry := heap.Pop(&edf.items).(*edfEntry)
	edf.currentTime = entry.deadline
	weight := weightFunc(entry.item)
	// update the index、deadline and put into priorityQueue again
	entry.deadline = entry.deadline + 1.0/weight
	entry.weight = weight
	entry.queuedTime = time.Now()
	heap.Push(&edf.items, entry)
	return entry.item
}

func (pq PriorityQueue) Len() int { return len(pq) }

// Less make us always pop the ones with the smallest deadline
//or the ones with a smaller index when the deadline is the same
func (pq PriorityQueue) Less(i, j int) bool {
	if pq[i].deadline == pq[j].deadline {
		return pq[i].queuedTime.Before(pq[j].queuedTime)
	}
	return pq[i].deadline < pq[j].deadline
}
func (pq PriorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*edfEntry))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	*pq = old[0 : len(old)-1]
	return old[len(old)-1]
}
