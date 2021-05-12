package mosn

import (
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mosn"
)

type MosnWrapper struct {
	m *mosn.Mosn
}

// This is a wrapper for main
func NewMosn(c *v2.MOSNConfig) *MosnWrapper {
	mosn.DefaultInitStage(c)
	m := mosn.NewMosn(c)
	return &MosnWrapper{
		m: m,
	}
}

// see details in cmd/mosn/main/control.go
func (m *MosnWrapper) Start() {
	mosn.DefaultPreStartStage(m.m)
	mosn.DefaultStartStage(m.m)
	m.m.Start()
}

func (m *MosnWrapper) Close() {
	m.m.Close()
}
