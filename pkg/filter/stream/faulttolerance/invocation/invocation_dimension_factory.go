package invocation

import (
	"mosn.io/api"
)

var dimensionNewFunc NewDimension

type NewDimension func(info api.RequestInfo) InvocationDimension

func RegisterNewDimension(newDimension NewDimension) {
	dimensionNewFunc = newDimension
}
