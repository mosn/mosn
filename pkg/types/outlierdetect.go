package types

type Detector interface {
	AddChangedStateCb(cb func(host Host))

	SuccessRateAverage() float64

	SuccessRateEjectionThreshold() float64
}

type DetectorHostMonitor interface {
}
