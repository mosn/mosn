package mhttp2

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/module/http2"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// types.Encoder
// types.Decoder
type serverCodec struct {
	sc      *http2.MServerConn
	preface bool
	init    bool
}

func (c *serverCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	ms := model.(*http2.MStream)
	err := ms.SendResponse()
	return nil, err
}

func (c *serverCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	if !c.init {
		c.init = true
		c.sc.Init()
	}
	if !c.preface {
		if err := c.sc.Framer.ReadPreface(data); err == nil {
			c.preface = true
		} else {
			return nil, err
		}
	}
	frame, _, err := c.sc.Framer.ReadFrame(ctx, data, 0)
	return frame, err
}

type clientCodec struct {
	cc *http2.MClientConn
}

func (c *clientCodec) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	ms := model.(*http2.MClientStream)
	err := ms.RoundTrip(ctx)
	return nil, err
}

func (c *clientCodec) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	frame, _, err := c.cc.Framer.ReadFrame(ctx, data, 0)
	return frame, err
}
