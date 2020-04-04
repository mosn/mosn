package faulttolerance

type InvocationStatDimension interface {
	GetMeasureKey() string
	GetInvocationKey() string
}
