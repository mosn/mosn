package tars

import (
	"context"

	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"mosn.io/mosn/pkg/types"
)

func decodeRequest(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	req := &Request{
		rawData: data.Bytes(),
		data:    data,
	}
	reqPacket := &requestf.RequestPacket{}
	is := codec.NewReader(data.Bytes())
	err = reqPacket.ReadFrom(is)
	if err != nil {
		return nil, err
	}
	// service aware
	metaHeader, err := getServiceAwareMeta(req)
	metaMap := &MetaMap{
		meta: metaHeader,
	}
	req.metaMap = metaMap
	req.cmd = reqPacket
	return req, nil
}

func decodeResponse(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	resp := &Response{
		metaMap: nil,
		rawData: data.Bytes(),
		data:    data,
	}
	respPacket := &requestf.ResponsePacket{}
	is := codec.NewReader(data.Bytes())
	err = respPacket.ReadFrom(is)
	if err != nil {
		return nil, err
	}
	resp.cmd = respPacket
	return resp, nil
}

func getServiceAwareMeta(request *Request) (map[string]string, error) {
	meta := make(map[string]string, 0)
	meta[ServiceNameHeader] = request.cmd.SServantName
	meta[MethodNameHeader] = request.cmd.SFuncName
	return meta, nil
}
