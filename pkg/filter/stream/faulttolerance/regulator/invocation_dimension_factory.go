package regulator

import (
	"mosn.io/api"
)

type NewDimension func(info api.RequestInfo) InvocationDimension

var dimensionNewFunc NewDimension

func RegisterNewDimension(newDimension NewDimension) {
	dimensionNewFunc = newDimension
}

func GetNewDimensionFunc() NewDimension {
	return dimensionNewFunc
}
