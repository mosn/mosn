package tars

import (
	"context"

	"github.com/TarsCloud/TarsGo/tars"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"github.com/juju/errors"
	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
)

func decodeRequest(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	frameLen, status := tars.TarsRequest(data.Bytes())
	if status != tars.PACKAGE_FULL {
		return nil, errors.New("tars request status fail")
	}
	req := &Request{}
	req.rawData = data.Bytes()[:frameLen]
	req.data = buffer.NewIoBufferBytes(req.rawData)
	reqPacket := &requestf.RequestPacket{}
	is := codec.NewReader(data.Bytes())
	err = reqPacket.ReadFrom(is)
	if err != nil {
		return nil, err
	}
	// service aware
	metaHeader, err := getServiceAwareMeta(req)
	for k, v := range metaHeader {
		req.Set(k, v)
	}
	req.cmd = reqPacket
	data.Drain(frameLen)
	return req, nil
}

func decodeResponse(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	frameLen, status := tars.TarsRequest(data.Bytes())
	if status != tars.PACKAGE_FULL {
		return nil, errors.New("tars request status fail")
	}
	resp := &Response{}
	resp.rawData = data.Bytes()[:frameLen]
	resp.data = buffer.NewIoBufferBytes(resp.rawData)
	respPacket := &requestf.ResponsePacket{}
	is := codec.NewReader(data.Bytes())
	err = respPacket.ReadFrom(is)
	if err != nil {
		return nil, err
	}
	resp.cmd = respPacket
	data.Drain(frameLen)
	return resp, nil
}

func getServiceAwareMeta(request *Request) (map[string]string, error) {
	meta := make(map[string]string, 0)
	meta[ServiceNameHeader] = request.cmd.SServantName
	meta[MethodNameHeader] = request.cmd.SFuncName
	return meta, nil
}
