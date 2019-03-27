package shm

type MetricsZone interface {
	GetEntry(name string) (*metricsEntry, error)

	AllocEntry(name string) (*metricsEntry, error)

	Free()
}
