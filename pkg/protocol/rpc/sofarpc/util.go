package sofarpc

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/serialize"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

// NewResponse build sofa response msg according to original cmd and respStatus
func NewResponse(ctx context.Context, cmd SofaRpcCmd, respStatus int16) (SofaRpcCmd, error) {
	switch c := cmd.(type) {
	case *BoltRequest:
		return &BoltResponse{
			Protocol:       c.Protocol,
			CmdType:        RESPONSE,
			CmdCode:        RPC_RESPONSE,
			Version:        c.Version,
			ReqID:          c.ReqID,
			Codec:          c.Codec,
			ResponseStatus: respStatus,
		}, nil
	case *BoltRequestV2:
		return &BoltResponseV2{
			BoltResponse: BoltResponse{
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

func Clone(ctx context.Context, origin SofaRpcCmd) (SofaRpcCmd, error) {
	//TODO: reuse req/resp struct
	//FIXME: comment clone logic, cause currently there was no need
	//switch c := origin.(type) {
	//case *BoltRequest:
	//	copy := &BoltRequest{}
	//	*copy = *c
	//	return copy, nil
	//case *BoltRequestV2:
	//	copy := &BoltRequestV2{}
	//	*copy = *c
	//	return copy, nil
	//case *BoltResponse:
	//	copy := &BoltResponse{}
	//	*copy = *c
	//	return copy, nil
	//case *BoltResponseV2:
	//	copy := &BoltResponseV2{}
	//	*copy = *cx
	//	return copy, nil
	//}
	//
	//log.ByContext(ctx).Errorf("[Prepare Error]Unknown model type")
	//return origin, rpc.ErrUnknownType
	return origin, nil
}

// NewBoltHeartbeat
// New Bolt Heartbeat with requestID as input
func NewBoltHeartbeat(requestID uint32) *BoltRequest {
	return &BoltRequest{
		Protocol: PROTOCOL_CODE_V1,
		CmdType:  REQUEST,
		CmdCode:  HEARTBEAT,
		Version:  1,
		ReqID:    requestID,
		Codec:    HESSIAN2_SERIALIZE, //todo: read default codec from config
		Timeout:  -1,
	}
}

// NewBoltHeartbeatAck
// New Bolt Heartbeat Ack with requestID as input
func NewBoltHeartbeatAck(requestID uint32) *BoltResponse {
	return &BoltResponse{
		Protocol:       PROTOCOL_CODE_V1,
		CmdType:        RESPONSE,
		CmdCode:        HEARTBEAT,
		Version:        1,
		ReqID:          requestID,
		Codec:          HESSIAN2_SERIALIZE, //todo: read default codec from config
		ResponseStatus: RESPONSE_STATUS_SUCCESS,
	}
}

func DeserializeBoltRequest(ctx context.Context, request *BoltRequest) {
	//get instance
	serializeIns := serialize.Instance

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	request.RequestHeader = protocolCtx.GetReqHeaders()

	//request.RequestHeader = make(map[string]string, 8)

	//logger
	logger := log.ByContext(ctx)

	//deserialize header
	serializeIns.DeSerialize(request.HeaderMap, &request.RequestHeader)
	logger.Debugf("Deserialize request header map:%v", request.RequestHeader)

	//deserialize class name
	serializeIns.DeSerialize(request.ClassName, &request.RequestClass)
	logger.Debugf("Request class name is:%s", request.RequestClass)
}

func DeserializeBoltResponse(ctx context.Context, response *BoltResponse) {
	//get instance
	serializeIns := serialize.Instance

	//logger
	logger := log.ByContext(ctx)

	protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	response.ResponseHeader = protocolCtx.GetRspHeaders()

	//response.ResponseHeader = make(map[string]string, 8)

	//deserialize header
	serializeIns.DeSerialize(response.HeaderMap, &response.ResponseHeader)
	logger.Debugf("Deserialize response header map: %+v", response.ResponseHeader)

	//deserialize class name
	serializeIns.DeSerialize(response.ClassName, &response.ResponseClass)
	logger.Debugf("Response ClassName is: %s", response.ResponseClass)
}
