package regulator

type InvocationDimension interface {
	GetInvocationKey() string
	GetMeasureKey() string
}
