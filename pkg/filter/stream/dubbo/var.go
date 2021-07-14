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
		variable.NewStringVariable(VarDubboRequestService, nil, nil, variable.DefaultStringSetter, 0),
		variable.NewStringVariable(VarDubboRequestMethod, nil, nil, variable.DefaultStringSetter, 0),
	}
)

func Init(conf map[string]interface{}) {
	// variable must registry
	for idx := range buildinVariables {
		variable.Register(buildinVariables[idx])
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
