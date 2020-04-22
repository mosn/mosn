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

	ele := edfScheduler.Next()
	assert.Equal(t, A, ele)
	ele = edfScheduler.Next()
	assert.Equal(t, C, ele)
	ele = edfScheduler.Next()
	assert.Equal(t, B, ele)

}
