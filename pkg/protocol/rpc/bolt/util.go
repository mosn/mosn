package bolt

import (
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"context"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

// NewResponse build sofa response msg according to original cmd and respStatus
func NewResponse(ctx context.Context, cmd rpc.SofaRpcCmd, respStatus int16) (rpc.SofaRpcCmd, error) {
	switch c := cmd.(type) {
	case *Request:
		return &Response{
			Protocol:       c.Protocol,
			CmdType:        RESPONSE,
			CmdCode:        RPC_RESPONSE,
			Version:        c.Version,
			ReqID:          c.ReqID,
			Codec:          c.Codec,
			ResponseStatus: respStatus,
		}, nil
	case *RequestV2:
		return &ResponseV2{
			Response: Response{
				Protocol:       c.Protocol,
				CmdType:        RESPONSE,
				CmdCode:        RPC_RESPONSE,
				Version:        c.Version,
				ReqID:          c.ReqID,
				Codec:          c.Codec,
				ResponseStatus: respStatus,
			},
			Version1:   c.Version1,
			SwitchCode: c.SwitchCode,
		}, nil
	}

	log.ByContext(ctx).Errorf("[NewResponse Error]Unknown model type")
	return cmd, rpc.ErrUnknownType
}

func Prepare(ctx context.Context, origin rpc.SofaRpcCmd) (rpc.SofaRpcCmd, error) {
	//TODO: reuse req/resp struct
	switch c := origin.(type) {
	case *Request:
		copy := &Request{}
		*copy = *c
		return copy, nil
	case *RequestV2:
		copy := &RequestV2{}
		*copy = *c
		return copy, nil
	case *Response:
		copy := &Response{}
		*copy = *c
		return copy, nil
	case *ResponseV2:
		copy := &ResponseV2{}
		*copy = *c
		return copy, nil
	}

	log.ByContext(ctx).Errorf("[Prepare Error]Unknown model type")
	return origin, rpc.ErrUnknownType
}

// NewHeartbeat
// New Bolt Heartbeat with requestID as input
func NewHeartbeat(requestID uint32) *Request {
	return &Request{
		Protocol: PROTOCOL_CODE_V1,
		CmdType:  REQUEST,
		CmdCode:  HEARTBEAT,
		Version:  1,
		ReqID:    requestID,
		Codec:    HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}
}

// NewHeartbeatAck
// New Heartbeat Ack with requestID as input
func NewHeartbeatAck(requestID uint32) *Response {
	return &Response{
		Protocol:       PROTOCOL_CODE_V1,
		CmdType:        RESPONSE,
		CmdCode:        HEARTBEAT,
		Version:        1,
		ReqID:          requestID,
		Codec:          HESSIAN2_SERIALIZE, //todo: read default codec from config
		ResponseStatus: RESPONSE_STATUS_SUCCESS,
	}
}

func DeserializeRequest(ctx context.Context, request *Request) {
	//get instance
	serializeIns := serialize.Instance

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	request.RequestHeader = protocolCtx.GetReqHeaders()

	//logger
	logger := log.ByContext(ctx)

	//deserialize header
	serializeIns.DeSerialize(request.HeaderMap, &request.RequestHeader)
	logger.Debugf("Deserialize request header map:%v", request.RequestHeader)

	//deserialize class name
	serializeIns.DeSerialize(request.ClassName, &request.RequestClass)
	logger.Debugf("Request class name is:%s", request.RequestClass)
}

func DeserializeResponse(ctx context.Context, response *Response) {
	//get instance
	serializeIns := serialize.Instance

	//logger
	logger := log.ByContext(ctx)

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	response.ResponseHeader = protocolCtx.GetRspHeaders()

	//deserialize header
	serializeIns.DeSerialize(response.HeaderMap, &response.ResponseHeader)
	logger.Debugf("Deserialize response header map: %+v", response.ResponseHeader)

	//deserialize class name
	serializeIns.DeSerialize(response.ClassName, &response.ResponseClass)
	logger.Debugf("Response ClassName is: %s", response.ResponseClass)
}
