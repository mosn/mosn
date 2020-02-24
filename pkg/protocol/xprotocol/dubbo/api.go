package dubbo

import (
	"mosn.io/mosn/pkg/types"
)

// NewRpcRequest is a utility function which build rpc Request object of bolt protocol.
func NewRpcRequest(headers types.HeaderMap, data types.IoBuffer) *Frame {
	frame, err := decodeFrame(nil, data)
	if err != nil {
		return nil
	}
	request, ok := frame.(*Frame)
	if !ok {
		return nil
	}

	// set headers
	if headers != nil {
		headers.Range(func(key, value string) bool {
			request.Set(key, value)
			return true
		})
	}
	return request
}

// NewRpcResponse is a utility function which build rpc Response object of bolt protocol.
func NewRpcResponse(headers types.HeaderMap, data types.IoBuffer) *Frame {
	frame, err := decodeFrame(nil, data)
	if err != nil {
		return nil
	}
	response, ok := frame.(*Frame)
	if !ok {
		return nil
	}

	// set headers
	if headers != nil {
		headers.Range(func(key, value string) bool {
			response.Set(key, value)
			return true
		})
	}
	return response
}
