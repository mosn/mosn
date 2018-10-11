package v1

import (
	"reflect"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/bolt"
	"strconv"
	"context"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"errors"
)

// PropertyHeaders map the cmdkey and its data type
var (
	PropertyHeaders = make(map[string]reflect.Kind, 11)
	boltv1          = new(boltv1conv)
)

func init() {
	PropertyHeaders[bolt.HeaderProtocolCode] = reflect.Uint8
	PropertyHeaders[bolt.HeaderCmdType] = reflect.Uint8
	PropertyHeaders[bolt.HeaderCmdCode] = reflect.Int16
	PropertyHeaders[bolt.HeaderVersion] = reflect.Uint8
	PropertyHeaders[bolt.HeaderReqID] = reflect.Uint32
	PropertyHeaders[bolt.HeaderCodec] = reflect.Uint8
	PropertyHeaders[bolt.HeaderClassLen] = reflect.Int16
	PropertyHeaders[bolt.HeaderHeaderLen] = reflect.Int16
	PropertyHeaders[bolt.HeaderContentLen] = reflect.Int
	PropertyHeaders[bolt.HeaderTimeout] = reflect.Int
	PropertyHeaders[bolt.HeaderRespStatus] = reflect.Int16
	PropertyHeaders[bolt.HeaderRespTimeMills] = reflect.Int64

	sofarpc.RegisterConv(bolt.PROTOCOL_CODE_V1, boltv1)
}

type boltv1conv struct{}

func (b *boltv1conv) MapToCmd(ctx context.Context, headers map[string]string) (rpc.SofaRpcCmd, error) {
	if len(headers) < 8 {
		return nil, errors.New("headers count not enough")
	}

	value := sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderProtocolCode)
	protocolCode := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderCmdType)
	cmdType := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderCmdCode)
	cmdCode := sofarpc.ConvertPropertyValueInt16(value)
	value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderVersion)
	version := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderReqID)
	requestID := sofarpc.ConvertPropertyValueUint32(value)
	value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderCodec)
	codec := sofarpc.ConvertPropertyValueUint8(value)
	//value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, sofarpc.HeaderClassLen)
	//classLength := sofarpc.ConvertPropertyValueInt16(value)
	//value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, sofarpc.HeaderHeaderLen)
	//headerLength := sofarpc.ConvertPropertyValueInt16(value)
	value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderContentLen)
	contentLength := sofarpc.ConvertPropertyValueInt(value)

	//class
	className := sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderClassName)

	//RPC Request
	if cmdType == bolt.REQUEST || cmdType == bolt.REQUEST_ONEWAY {
		value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderTimeout)
		timeout := sofarpc.ConvertPropertyValueInt(value)

		sofabuffers := SofaProtocolBuffersByContext(ctx)
		request := &sofabuffers.BoltEncodeReq
		request.Protocol = protocolCode
		request.CmdType = cmdType
		request.CmdCode = cmdCode
		request.Version = version
		request.ReqID = requestID
		request.Codec = codec
		request.Timeout = timeout
		//request.ClassLen = classLength
		//request.HeaderLen = headerLength
		request.ContentLen = contentLength
		request.RequestClass = className
		request.RequestHeader = headers
		return request, nil
	} else if cmdType == bolt.RESPONSE {
		value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderRespStatus)
		responseStatus := sofarpc.ConvertPropertyValueInt16(value)
		value = sofarpc.GetPropertyValue1(PropertyHeaders, headers, bolt.HeaderRespTimeMills)
		responseTime := sofarpc.ConvertPropertyValueInt64(value)

		sofabuffers := SofaProtocolBuffersByContext(ctx)
		response := &sofabuffers.BoltEncodeRsp
		response.Protocol = protocolCode
		response.CmdType = cmdType
		response.CmdCode = cmdCode
		response.Version = version
		response.ReqID = requestID
		response.Codec = codec
		response.ResponseStatus = responseStatus
		//response.ClassLen = classLength
		//response.HeaderLen = headerLength
		response.ContentLen = contentLength
		response.ResponseClass = className
		response.ResponseHeader = headers
		response.ResponseTimeMillis = responseTime
		return response, nil
	}

	return nil, rpc.ErrUnknownType
}

//Convert BoltV1's Protocol Header  and Content Header to Map[string]string
func (b *boltv1conv) MapToFields(ctx context.Context, cmd rpc.SofaRpcCmd) (map[string]string, error) {
	switch c := cmd.(type) {
	case *bolt.Request:
		return mapReqToFields(ctx, c)
	case *bolt.Response:
		return mapRespToFields(ctx, c)
	}

	return nil, rpc.ErrUnknownType
}

func mapReqToFields(ctx context.Context, req *bolt.Request) (map[string]string, error) {
	// TODO: map reuse
	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	//headers := make(map[string]string, 9+len(req.RequestHeader))
	headers := req.RequestHeader

	headers[sofarpc.SofaPropertyHeader(bolt.HeaderProtocolCode)] = strconv.FormatUint(uint64(req.Protocol), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderCmdType)] = strconv.FormatUint(uint64(req.CmdType), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderCmdCode)] = strconv.FormatUint(uint64(req.CmdCode), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderVersion)] = strconv.FormatUint(uint64(req.Version), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderReqID)] = strconv.FormatUint(uint64(req.ReqID), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderCodec)] = strconv.FormatUint(uint64(req.Codec), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderTimeout)] = strconv.FormatUint(uint64(req.Timeout), 10)

	// TODO: bypass length header
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(req.ClassLen), 10)
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(req.HeaderLen), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderContentLen)] = strconv.FormatUint(uint64(req.ContentLen), 10)

	headers[sofarpc.SofaPropertyHeader(bolt.HeaderClassName)] = req.RequestClass

	return headers, nil
}

func mapRespToFields(ctx context.Context, resp *bolt.Response) (map[string]string, error) {
	// TODO: map reuse
	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	//headers := make(map[string]string, 12)

	headers := resp.ResponseHeader

	headers[sofarpc.SofaPropertyHeader(bolt.HeaderProtocolCode)] = strconv.FormatUint(uint64(resp.Protocol), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderCmdType)] = strconv.FormatUint(uint64(resp.CmdType), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderCmdCode)] = strconv.FormatUint(uint64(resp.CmdCode), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderVersion)] = strconv.FormatUint(uint64(resp.Version), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderReqID)] = strconv.FormatUint(uint64(resp.ReqID), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderCodec)] = strconv.FormatUint(uint64(resp.Codec), 10)

	// TODO: bypass length header
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(resp.ClassLen), 10)
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(resp.HeaderLen), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderContentLen)] = strconv.FormatUint(uint64(resp.ContentLen), 10)

	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderRespStatus)] = strconv.FormatUint(uint64(resp.ResponseStatus), 10)
	headers[sofarpc.SofaPropertyHeader(bolt.HeaderRespTimeMills)] = strconv.FormatUint(uint64(resp.ResponseTimeMillis), 10)

	headers[sofarpc.SofaPropertyHeader(bolt.HeaderClassName)] = resp.ResponseClass

	return headers, nil
}
