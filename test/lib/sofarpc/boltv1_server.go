package sofarpc

import (
	"context"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

type BoltV1ResponseBuilder struct {
	Status  int16
	Header  map[string]string
	Content types.IoBuffer
}

func (b *BoltV1ResponseBuilder) Build(req *bolt.Request) *bolt.Response {
	reqId := uint32(req.GetRequestId())
	cmd := bolt.NewRpcResponse(reqId, uint16(b.Status), protocol.CommonHeader(b.Header), nil)
	if b.Content != nil {
		cmd.Content = b.Content
	}
	return cmd
}

var DefaultBuilder = &BoltV1ResponseBuilder{
	Status: int16(bolt.ResponseStatusSuccess),
	Header: map[string]string{
		"mosn-test-default": "boltv1",
	},
	Content: buffer.NewIoBufferString("default-boltv1"),
}

var ErrorBuilder = &BoltV1ResponseBuilder{
	Status: int16(bolt.ResponseStatusError),
	Header: map[string]string{
		"error-message": "no matched config",
	},
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

func (s *BoltV1Serve) MakeResponse(req *bolt.Request) *bolt.Response {
	header := make(map[string]string)
	req.GetHeader().Range(func(key, value string) bool {
		header[key] = value
		return true
	})
	for _, cfg := range s.Configs {
		if cfg.Match(header) {
			return cfg.Builder.Build(req)
		}
	}
	return ErrorBuilder.Build(req)
}

func (s *BoltV1Serve) Serve(reqdata types.IoBuffer) *WriteResponseData {
	ctx := context.Background()
	engine := xprotocol.GetProtocol(bolt.ProtocolName)
	cmd, err := engine.Decode(ctx, reqdata)
	if cmd == nil || err != nil {
		return nil
	}
	var status int16
	var data []byte
	if req, ok := cmd.(*bolt.Request); ok {
		var resp xprotocol.XRespFrame
		switch req.CmdCode {
		case bolt.CmdCodeHeartbeat:
			resp = engine.Reply(req)
			status = int16(bolt.ResponseStatusSuccess) // heartbeat must be success
		case bolt.CmdCodeRpcRequest:
			resp = s.MakeResponse(req)
			if s, ok := resp.(*bolt.Response); ok {
				status = int16(s.GetStatusCode())
			}
		default:
			return nil
		}
		if resp != nil {
			// header
			iobuf, err := engine.Encode(ctx, resp)
			if err == nil {
				data = iobuf.Bytes()
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
