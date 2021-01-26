package dubbo

import (
	"context"
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/pkg/buffer"
	"testing"
)

// decoded result should equal to request
func TestEncode(t *testing.T) {
	var (
		ctx            = context.TODO()
		originalHeader = protocol.CommonHeader{
			"k1": "v1",
			"k2": "v2",
		}
	)

	//////////// request part start
	req := NewRpcRequest(originalHeader,
		buffer.NewIoBufferBytes(buildDubboRequest(111)))

	assert.NotNil(t, req)

	_, err := encodeRequest(ctx, req)
	assert.Nil(t, err)
	//////////// request part end

	//////////// response part start
	rsp := NewRpcResponse(originalHeader,
		buffer.NewIoBufferBytes(buildDubboResponse(222)))

	assert.NotNil(t, rsp)

	_, err = encodeResponse(ctx, rsp)
	assert.Nil(t, err)
	//////////// response part end
}

func buildDubboRequest(requestId uint64) []byte {
	service := hessian.Service{
		Path:      "com.alipay.test",
		Interface: "test",
		Group:     "test",
		Version:   "v1",
		Method:    "testCall",
	}
	codec := hessian.NewHessianCodec(nil)
	header := hessian.DubboHeader{
		SerialID: 2,
		Type:     hessian.PackageRequest,
		ID:       int64(requestId),
	}
	body := hessian.NewRequest([]interface{}{}, nil)
	reqData, err := codec.Write(service, header, body)
	if err != nil {
		return nil
	}
	return reqData
}

func buildDubboResponse(requestId uint64) []byte {
	service := hessian.Service{
		Path:      "com.alipay.test",
		Interface: "test",
		Group:     "test",
		Version:   "v1",
		Method:    "testCall",
	}
	codec := hessian.NewHessianCodec(nil)
	header := hessian.DubboHeader{
		SerialID:       2,
		Type:           hessian.PackageResponse,
		ID:             int64(requestId),
		ResponseStatus: uint8(ResponseStatusSuccess),
	}
	body := hessian.NewResponse(nil, nil, nil)
	reqData, err := codec.Write(service, header, body)
	if err != nil {
		return nil
	}
	return reqData
}
