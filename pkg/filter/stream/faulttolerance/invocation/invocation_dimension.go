package invocation

type InvocationDimension interface {
	GetInvocationKey() string
	GetMeasureKey() string
}
