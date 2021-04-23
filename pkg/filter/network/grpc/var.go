package grpc

import (
	"context"

	"mosn.io/api"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
)

const (
	grpcName    string = "gRPC"
	serviceName string = "service_name"
)

var (
	builtinVariables = []variable.Variable{
		variable.NewBasicVariable(grpcName+"_"+serviceName, serviceName, serviceNameGetter, nil, 0),
	}
)

func serviceNameGetter(ctx context.Context, value *variable.IndexedValue, data interface{}) (string, error) {
	headers, ok := mosnctx.Get(ctx, types.ContextKeyDownStreamHeaders).(api.HeaderMap)
	if !ok {
		return variable.ValueNotFound, nil
	}
	headerKey, ok := data.(string)
	if !ok {
		return variable.ValueNotFound, nil
	}

	serviceName, ok := headers.Get(headerKey)
	if ok {
		return serviceName, nil
	}
	return variable.ValueNotFound, nil
}

func init() {
	for idx := range builtinVariables {
		variable.RegisterVariable(builtinVariables[idx])
	}

	variable.RegisterProtocolResource(api.ProtocolName(grpcName), api.PATH, serviceName)
}
