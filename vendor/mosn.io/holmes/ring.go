package holmes

type ring struct {
	data   []int
	idx    int
	sum    int
	maxLen int
}

func newRing(maxLen int) ring {
	return ring{
		data:   make([]int, 0, maxLen),
		idx:    0,
		maxLen: maxLen,
	}
}

func (r *ring) push(i int) {
	if r.maxLen == 0 {
		return
	}

	// the first round
	if len(r.data) < r.maxLen {
		r.sum += i
		r.data = append(r.data, i)
		return
	}

	r.sum += i - r.data[r.idx]

	// the ring is expanded, just write to the position
	r.data[r.idx] = i
	r.idx = (r.idx + 1) % r.maxLen
}

func (r *ring) avg() int {
	// Check if the len(r.data) is zero before dividing
	if r.maxLen == 0 || len(r.data) == 0 {
		return 0
	}
	return r.sum / len(r.data)
}
