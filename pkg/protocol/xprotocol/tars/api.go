package tars

import "mosn.io/mosn/pkg/types"

// NewRpcRequest is a utility function which build rpc Request object of bolt protocol.
func NewRpcRequest(headers types.HeaderMap, data types.IoBuffer) *Request {
	frame, err := decodeRequest(nil, data)
	if err != nil {
		return nil
	}
	request, ok := frame.(*Request)
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
func NewRpcResponse(headers types.HeaderMap, data types.IoBuffer) *Response {
	frame, err := decodeResponse(nil, data)
	if err != nil {
		return nil
	}
	response, ok := frame.(*Response)
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
