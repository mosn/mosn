package invocation

type InvocationStat struct {
	dimension      InvocationDimension
	invocationKey  string
	measureKey     string
	callCount      int64
	exceptionCount int64
	uselessCycle   int32
	healthy        bool
	downgradeTime  int64
}
