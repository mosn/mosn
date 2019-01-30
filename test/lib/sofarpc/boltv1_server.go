package sofarpc

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type BoltV1ResponseBuilder struct {
	Status  int16
	Header  map[string]string
	Content types.IoBuffer
}

func (b *BoltV1ResponseBuilder) Build(req sofarpc.SofaRpcCmd) sofarpc.SofaRpcCmd {
	cmd := &sofarpc.BoltResponse{
		Protocol:       sofarpc.PROTOCOL_CODE_V1,
		CmdType:        sofarpc.RESPONSE,
		CmdCode:        sofarpc.RPC_RESPONSE,
		Version:        1,
		Codec:          sofarpc.HESSIAN2_SERIALIZE,
		ResponseStatus: b.Status,
		ResponseHeader: b.Header, // codec encode will encode it into header map byte
	}
	if b.Content != nil {
		cmd.Content = b.Content
		cmd.ContentLen = b.Content.Len()
	}
	return cmd
}

var DefaultBuilder = &BoltV1ResponseBuilder{
	Status: sofarpc.RESPONSE_STATUS_SUCCESS,
	Header: map[string]string{
		"mosn-test-default": "boltv1",
	},
	Content: buffer.NewIoBufferString("default-boltv1"),
}

// TODO: Support More
type BoltV1ReponseConfig struct {
	ExpectedHeader      map[string]string
	UnexpectedHeaderKey []string
	Builder             *BoltV1ResponseBuilder
}

func (cfg *BoltV1ReponseConfig) Match(header map[string]string) bool {
	if len(header) < len(cfg.ExpectedHeader) {
		return false
	}
	for key, value := range cfg.ExpectedHeader {
		if header[key] != value {
			return false
		}
	}
	for _, key := range cfg.UnexpectedHeaderKey {
		if _, ok := header[key]; ok {
			return false
		}
	}
	return true
}

type BoltV1Serve struct {
	Configs []*BoltV1ReponseConfig
}

func (s *BoltV1Serve) MakeResponse(req *sofarpc.BoltRequest) sofarpc.SofaRpcCmd {
	header := req.Header()
	for _, cfg := range s.Configs {
		if cfg.Match(header) {
			return cfg.Builder.Build(req)
		}
	}
	return nil
}

func (s *BoltV1Serve) Serve(reqdata types.IoBuffer) *WriteResponseData {
	ctx := context.Background()
	cmd, _ := codec.BoltCodec.Decode(ctx, reqdata)
	if cmd == nil {
		return nil
	}
	var status int16
	var data []byte
	if req, ok := cmd.(*sofarpc.BoltRequest); ok {
		var resp sofarpc.SofaRpcCmd
		switch req.CommandCode() {
		case sofarpc.HEARTBEAT:
			resp = codec.BoltCodec.Reply()
			status = sofarpc.RESPONSE_STATUS_SUCCESS // heartbeat must be success
		case sofarpc.RPC_REQUEST:
			resp = s.MakeResponse(req)
		default:
			return nil
		}
		if resp != nil {
			resp.SetRequestID(req.RequestID())
			// header
			iobuf, err := codec.BoltCodec.Encode(ctx, resp)
			if err == nil {
				data = iobuf.Bytes()
			}
			// body
			if resp.Data() != nil {
				data = append(data, resp.Data().Bytes()...)
			}
			return &WriteResponseData{
				Status:      status,
				DataToWrite: data,
			}
		}
	}
	return nil

}

var DefaultBoltV1Serve = &BoltV1Serve{
	Configs: []*BoltV1ReponseConfig{
		{
			Builder: DefaultBuilder,
		},
	},
}
