package protocol

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"context"
	"errors"
)

func init() {
	identity := new(identity)

	RegisterConv(HTTP1, HTTP2, identity)
	RegisterConv(HTTP2, HTTP1, identity)
}

type ProtocolConv interface {
	ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error)

	ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error)

	ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error)
}

var (
	protoConvFactory = make(map[types.Protocol]map[types.Protocol]ProtocolConv)

	ErrNotFound = errors.New("no convert function found for given protocol pair")
)

func RegisterConv(src, dst types.Protocol, f ProtocolConv) {
	if _, subOk := protoConvFactory[src]; !subOk {
		protoConvFactory[src] = make(map[types.Protocol]ProtocolConv)
	}

	protoConvFactory[src][dst] = f
}

func ConvertHeader(ctx context.Context, src, dst types.Protocol, srcHeader types.HeaderMap) (types.HeaderMap, error) {
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvHeader(ctx, srcHeader)
		}
	}
	return nil, ErrNotFound
}

func ConvertData(ctx context.Context, src, dst types.Protocol, srcData types.IoBuffer) (types.IoBuffer, error) {
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvData(ctx, srcData)
		}
	}
	return nil, ErrNotFound
}

func ConvertTrailer(ctx context.Context, src, dst types.Protocol, srcTrailer types.HeaderMap) (types.HeaderMap, error) {
	if sub, subOk := protoConvFactory[src]; subOk {
		if f, ok := sub[dst]; ok {
			return f.ConvTrailer(ctx, srcTrailer)
		}
	}
	return nil, ErrNotFound
}

// identity
type identity struct{}

func (c *identity) ConvHeader(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}

func (c *identity) ConvData(ctx context.Context, buffer types.IoBuffer) (types.IoBuffer, error) {
	return buffer, nil
}

func (c *identity) ConvTrailer(ctx context.Context, headerMap types.HeaderMap) (types.HeaderMap, error) {
	return headerMap, nil
}
