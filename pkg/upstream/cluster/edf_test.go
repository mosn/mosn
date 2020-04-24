package cluster

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test(t *testing.T) {
	A := &mockHost{name: "A", weight: 4}
	B := &mockHost{name: "B", weight: 2}
	C := &mockHost{name: "C", weight: 3}
	D := &mockHost{name: "D", weight: 1}

	edfScheduler := newEdfScheduler(4)
	edfScheduler.Add(A, float64(A.weight))
	edfScheduler.Add(B, float64(B.weight))
	edfScheduler.Add(C, float64(C.weight))
	edfScheduler.Add(D, float64(D.weight))
	weightFunc := func(item WeightItem) float64 {
		return float64(item.Weight())
	}
	ele := edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, A, ele)
	ele = edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, C, ele)

}
