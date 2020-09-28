package dubbo

import (
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

var (
	buildinVariables = []variable.Variable{
		variable.NewIndexedVariable(types.VarDubboRequestService, nil, nil, variable.BasicSetter, 0),
		variable.NewIndexedVariable(types.VarDubboRequestMethod, nil, nil, variable.BasicSetter, 0),
	}
)

func Init() {
	// variable must registry
	for idx := range buildinVariables {
		variable.RegisterVariable(buildinVariables[idx])
	}
}
