package regulator

import (
	"mosn.io/api"
)

func init() {
	// register default dimension
	RegisterNewDimension(defaultDimensionFunc)
}

type NewDimension func(info api.RequestInfo) InvocationDimension

var dimensionNewFunc NewDimension

func RegisterNewDimension(newDimension NewDimension) {
	dimensionNewFunc = newDimension
}

func GetNewDimensionFunc() NewDimension {
	return dimensionNewFunc
}

type defaultDimension struct {
	invocationKey string
	measureKey    string
}

func (d *defaultDimension) GetInvocationKey() string {
	return d.invocationKey
}

func (d *defaultDimension) GetMeasureKey() string {
	return d.measureKey
}

func defaultDimensionFunc(info api.RequestInfo) InvocationDimension {
	if info == nil || info.UpstreamHost() == nil || info.RouteEntry() == nil {
		return &defaultDimension{}
	} else {
		return &defaultDimension{
			invocationKey: info.UpstreamHost().AddressString(),
			measureKey:    info.RouteEntry().ClusterName(),
		}
	}
}
