package mosn

import (
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mosn"
	_ "mosn.io/mosn/pkg/server/keeper"
)

type MosnWrapper struct {
	m *mosn.Mosn
}

// This is a wrapper for main
func NewMosn(c *v2.MOSNConfig) *MosnWrapper {
	mosn.DefaultInitStage(c)
	m := mosn.NewMosn()
	m.Init(c)
	return &MosnWrapper{
		m: m,
	}
}

// see details in cmd/mosn/main/control.go
func (m *MosnWrapper) Start() {
	mosn.DefaultPreStartStage(m.m)
	mosn.DefaultStartStage(m.m)
	m.m.Start()
	m.m.InheritConnections()
}

func (m *MosnWrapper) Close() {
	m.m.Close(false)
}
