package bolt

import "mosn.io/mosn/pkg/types"

// NewRpcRequest is a utility function which build rpc Request object of bolt protocol.
func NewRpcRequest(requestId uint32, headers types.HeaderMap, data types.IoBuffer) *Request {
	request := &Request{
		RequestHeader: RequestHeader{
			Protocol:  ProtocolCode,
			CmdType:   CmdTypeRequest,
			CmdCode:   CmdCodeRpcRequest,
			Version:   ProtocolVersion,
			RequestId: requestId,
			Codec:     Hessian2Serialize,
			Timeout:   -1,
		},
	}

	// set headers
	headers.Range(func(key, value string) bool {
		request.Set(key, value)
		return true
	})

	// set content
	request.Content = data

	return request
}

// NewRpcResponse is a utility function which build rpc Response object of bolt protocol.
func NewRpcResponse(requestId uint32, statusCode uint16, headers types.HeaderMap, data types.IoBuffer) *Response{
	response := &Response{
		ResponseHeader: ResponseHeader{
			Protocol:       ProtocolCode,
			CmdType:        CmdTypeResponse,
			CmdCode:        CmdCodeRpcResponse,
			Version:        ProtocolVersion,
			RequestId:      requestId,
			Codec:          Hessian2Serialize,
			ResponseStatus: statusCode,
		},
	}

	// set headers
	headers.Range(func(key, value string) bool {
		response.Set(key, value)
		return true
	})

	// set content
	response.Content = data

	return response
}
