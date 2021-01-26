package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	A := &mockHost{name: "A", w: 4}
	B := &mockHost{name: "B", w: 2}
	C := &mockHost{name: "C", w: 3}
	D := &mockHost{name: "D", w: 1}

	edfScheduler := newEdfScheduler(4)
	edfScheduler.Add(A, float64(A.w))
	edfScheduler.Add(B, float64(B.w))
	edfScheduler.Add(C, float64(C.w))
	edfScheduler.Add(D, float64(D.w))
	weightFunc := func(item WeightItem) float64 {
		return float64(item.Weight())
	}
	ele := edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, A, ele)
	ele = edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, C, ele)
	ele = edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, B, ele)
	ele = edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, A, ele)

}
