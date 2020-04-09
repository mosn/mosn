package faulttolerance

type InvocationDimension struct {
	dimension string
	address   string
}

var emptyInvocationDimension = InvocationDimension{
	dimension: "",
	address:   "",
}

func NewInvocationDimension(dimension string, address string) InvocationDimension {
	invocationDimension := InvocationDimension{
		dimension: dimension,
		address:   address,
	}
	return invocationDimension
}

func GetEmptyInvocationDimension() InvocationDimension {
	return emptyInvocationDimension
}
