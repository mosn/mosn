package router

import "math/rand"

type mirrorImpl struct {
	cluster        string
	numberator     int
	denominatorNum int
	rand           *rand.Rand
}

func (m *mirrorImpl) IsTrans() bool {
	return m.numberator <= (m.rand.Intn(m.denominatorNum) + 1)
}

func (m *mirrorImpl) Cluster() string {
	return m.cluster
}
