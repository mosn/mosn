package conv

import (
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"strconv"
	"reflect"
	"context"
	"github.com/kataras/iris/core/errors"
)

// BoltV1PropertyHeaders map the cmdkey and its data type
var (
	BoltV1PropertyHeaders = make(map[string]reflect.Kind, 11)
	boltv1 = new(boltv1conv)
)

func init() {
	BoltV1PropertyHeaders[sofarpc.HeaderProtocolCode] = reflect.Uint8
	BoltV1PropertyHeaders[sofarpc.HeaderCmdType] = reflect.Uint8
	BoltV1PropertyHeaders[sofarpc.HeaderCmdCode] = reflect.Int16
	BoltV1PropertyHeaders[sofarpc.HeaderVersion] = reflect.Uint8
	BoltV1PropertyHeaders[sofarpc.HeaderReqID] = reflect.Uint32
	BoltV1PropertyHeaders[sofarpc.HeaderCodec] = reflect.Uint8
	BoltV1PropertyHeaders[sofarpc.HeaderClassLen] = reflect.Int16
	BoltV1PropertyHeaders[sofarpc.HeaderHeaderLen] = reflect.Int16
	BoltV1PropertyHeaders[sofarpc.HeaderContentLen] = reflect.Int
	BoltV1PropertyHeaders[sofarpc.HeaderTimeout] = reflect.Int
	BoltV1PropertyHeaders[sofarpc.HeaderRespStatus] = reflect.Int16
	BoltV1PropertyHeaders[sofarpc.HeaderRespTimeMills] = reflect.Int64

	sofarpc.RegisterConv(sofarpc.PROTOCOL_CODE_V1, boltv1)
}

type boltv1conv struct{}

func (b *boltv1conv) MapToCmd(ctx context.Context, headers map[string]string) (sofarpc.ProtoBasicCmd, error) {
	if len(headers) < 10 {
		return nil, errors.New("headers count not enough")
	}

	value := sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderProtocolCode)
	protocolCode := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderCmdType)
	cmdType := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderCmdCode)
	cmdCode := sofarpc.ConvertPropertyValueInt16(value)
	value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderVersion)
	version := sofarpc.ConvertPropertyValueUint8(value)
	value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderReqID)
	requestID := sofarpc.ConvertPropertyValueUint32(value)
	value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderCodec)
	codec := sofarpc.ConvertPropertyValueUint8(value)
	//value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderClassLen)
	//classLength := sofarpc.ConvertPropertyValueInt16(value)
	//value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderHeaderLen)
	//headerLength := sofarpc.ConvertPropertyValueInt16(value)
	value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderContentLen)
	contentLength := sofarpc.ConvertPropertyValueInt(value)

	//class
	className := sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderClassName)

	//RPC Request
	if cmdType == sofarpc.REQUEST || cmdType == sofarpc.REQUEST_ONEWAY {
		value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderTimeout)
		timeout := sofarpc.ConvertPropertyValueInt(value)

		sofabuffers := sofarpc.SofaProtocolBuffersByContext(ctx)
		request := &sofabuffers.BoltEncodeReq
		request.Protocol = protocolCode
		request.CmdType = cmdType
		request.CmdCode = cmdCode
		request.Version = version
		request.ReqID = requestID
		request.CodecPro = codec
		request.Timeout = timeout
		//request.ClassLen = classLength
		//request.HeaderLen = headerLength
		request.ContentLen = contentLength
		request.RequestClass = className
		request.RequestHeader = headers
		return request, nil
	} else if cmdType == sofarpc.RESPONSE {
		//todo : review
		value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderRespStatus)
		responseStatus := sofarpc.ConvertPropertyValueInt16(value)
		value = sofarpc.GetPropertyValue1(BoltV1PropertyHeaders, headers, sofarpc.HeaderRespTimeMills)
		responseTime := sofarpc.ConvertPropertyValueInt64(value)

		sofabuffers := sofarpc.SofaProtocolBuffersByContext(ctx)
		response := &sofabuffers.BoltEncodeRsp
		response.Protocol = protocolCode
		response.CmdType = cmdType
		response.CmdCode = cmdCode
		response.Version = version
		response.ReqID = requestID
		response.CodecPro = codec
		response.ResponseStatus = responseStatus
		//response.ClassLen = classLength
		//response.HeaderLen = headerLength
		response.ContentLen = contentLength
		response.ResponseClass = className
		response.ResponseHeader = headers
		response.ResponseTimeMillis = responseTime
		return response, nil
	}

	return nil, errors.New(sofarpc.InvalidCommandType)
}

//Convert BoltV1's Protocol Header  and Content Header to Map[string]string
func (b *boltv1conv) MapToFields(ctx context.Context, cmd sofarpc.ProtoBasicCmd) (map[string]string, error) {
	switch c := cmd.(type) {
	case *sofarpc.BoltRequestCommand:
		return mapReqToFields(ctx, c)

	case *sofarpc.BoltResponseCommand:
		return mapRespToFields(ctx, c)
	}

	return nil, errors.New(sofarpc.InvalidCommandType)
}

func mapReqToFields(ctx context.Context, req *sofarpc.BoltRequestCommand) (map[string]string, error) {
	// TODO: map reuse
	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)

	headers := make(map[string]string, 11)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(req.Protocol), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(req.CmdType), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(req.CmdCode), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(req.Version), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(req.ReqID), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(req.CodecPro), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)] = strconv.FormatUint(uint64(req.Timeout), 10)

	// TODO: bypass length header
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(req.ClassLen), 10)
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(req.HeaderLen), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(req.ContentLen), 10)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = req.RequestClass
	return headers, nil
}

func mapRespToFields(ctx context.Context, resp *sofarpc.BoltResponseCommand) (map[string]string, error) {
	// TODO: map reuse
	//protocolCtx := protocol.ProtocolBuffersByContext(ctx)
	headers := make(map[string]string, 12)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(resp.Protocol), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(resp.CmdType), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(resp.CmdCode), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(resp.Version), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(resp.ReqID), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(resp.CodecPro), 10)

	// TODO: bypass length header
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(resp.ClassLen), 10)
	//headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(resp.HeaderLen), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(resp.ContentLen), 10)

	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)] = strconv.FormatUint(uint64(resp.ResponseStatus), 10)
	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespTimeMills)] = strconv.FormatUint(uint64(resp.ResponseTimeMillis), 10)

	headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = resp.ResponseClass

	return headers, nil
}
