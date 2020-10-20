package lib

import "mosn.io/mosn/test/lib/types"

type CreateMockServer func(config interface{}) types.MockServer

type CreateMockClient func(config interface{}) types.MockClient

// protoco: CreateFunc
var clientFactory map[string]CreateMockClient = map[string]CreateMockClient{}
var serverFactory map[string]CreateMockServer = map[string]CreateMockServer{}

func RegisterCreateClient(protocol string, f CreateMockClient) {
	clientFactory[protocol] = f
}

func RegisterCreateServer(protocol string, f CreateMockServer) {
	serverFactory[protocol] = f
}

func CreateServer(protocol string, config interface{}) types.MockServer {
	f, ok := serverFactory[protocol]
	if !ok {
		return nil
	}
	return f(config)
}

func CreateClient(protocol string, config interface{}) types.MockClient {
	f, ok := clientFactory[protocol]
	if !ok {
		return nil
	}
	return f(config)
}
