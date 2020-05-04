package regulator

type Regulator interface {
	Regulate(stat *InvocationStat)
}
