package faulttolerance

type FaultToleranceConfigManager struct {
}

func (m *FaultToleranceConfigManager) GetTimeWindow(dimension string) uint32 {
	return 0
}

func GetFaultToleranceConfigManagerInstance() *FaultToleranceConfigManager {
	return nil
}
