package dubbo

import (
	"mosn.io/mosn/pkg/variable"
)

// [Protocol]: dubbo
const (
	dubboProtocolName      = "Dubbo_"
	VarDubboRequestService = dubboProtocolName + "service"
	VarDubboRequestMethod  = dubboProtocolName + "method"
)

var (
	buildinVariables = []variable.Variable{
		variable.NewIndexedVariable(VarDubboRequestService, nil, nil, variable.BasicSetter, 0),
		variable.NewIndexedVariable(VarDubboRequestMethod, nil, nil, variable.BasicSetter, 0),
	}
)

func Init(conf map[string]interface{}) {
	// variable must registry
	for idx := range buildinVariables {
		variable.RegisterVariable(buildinVariables[idx])
	}

	// init subset key
	sskObj := conf[subsetKey]
	if sskObj == nil {
		return
	}

	ssk, ok := sskObj.(string)
	if !ok {
		return
	}
	podSubsetKey = ssk
	return
}
