package sofarpc

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"context"
	"github.com/kataras/iris/core/errors"
	"github.com/alipay/sofa-mosn/pkg/protocol"
)

var (
	sofaConvFactory = make(map[byte]SofaConv)

	//TODO universe definition
	ErrUnsupportedProtocol = errors.New(types.UnSupportedProCode)
	ErrNoProtocol          = errors.New(NoProCodeInHeader)
)

func init() {
	http2sofa := new(http2sofa)
	sofa2http := new(sofa2http)

	protocol.RegisterConv(protocol.HTTP1, SofaRPC, http2sofa)
	protocol.RegisterConv(protocol.HTTP2, SofaRPC, http2sofa)

	protocol.RegisterConv(SofaRPC, protocol.HTTP1, sofa2http)
	protocol.RegisterConv(SofaRPC, protocol.HTTP2, sofa2http)
}

type SofaConv interface {
	MapToCmd(ctx context.Context, headerMap map[string]string) (ProtoBasicCmd, error)

	MapToFields(ctx context.Context, cmd ProtoBasicCmd) (map[string]string, error)
}

func RegisterConv(protocol byte, conv SofaConv) {
	sofaConvFactory[protocol] = conv
}

// MapToCmd  expect src header data type as `protocol.CommonHeader`
func MapToCmd(ctx context.Context, headerMap map[string]string) (ProtoBasicCmd, error) {

	if proto, exist := headerMap[SofaPropertyHeader(HeaderProtocolCode)]; exist {
		protoValue := ConvertPropertyValueUint8(proto)
		protocolCode := protoValue

		if conv, ok := sofaConvFactory[protocolCode]; ok {
			return conv.MapToCmd(ctx, headerMap)
		}
		return nil, ErrUnsupportedProtocol
	}
	return nil, ErrNoProtocol
}

// MapToFields expect src header data type as `ProtoBasicCmd`
func MapToFields(ctx context.Context, cmd ProtoBasicCmd) (map[string]string, error) {
	protocol := cmd.GetProtocol()

	if conv, ok := sofaConvFactory[protocol]; ok {
		return conv.MapToFields(ctx, cmd)
	}
	return nil, ErrUnsupportedProtocol
}

// http/x -> sofarpc
type http2sofa struct{}

func (c *http2sofa) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	if header, ok := headerMap.(protocol.CommonHeader); ok {
		return MapToCmd(ctx, header)
	}
	return nil, errors.New("header type not supported")
}

func (c *http2sofa) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *http2sofa) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

// sofarpc -> http/x
type sofa2http struct{}

func (c *sofa2http) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	if cmd, ok := headerMap.(ProtoBasicCmd); ok {
		header, err := MapToFields(ctx, cmd)
		return protocol.CommonHeader(header), err
	}
	return nil, errors.New("header type not supported")
}

func (c *sofa2http) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *sofa2http) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}
