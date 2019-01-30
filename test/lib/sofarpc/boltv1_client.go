package sofarpc

import (
	"bytes"
	"fmt"
	"time"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func BuildBoltV1Request(id uint64, header map[string]string, body []byte) (types.HeaderMap, types.IoBuffer) {
	cmd := &sofarpc.BoltRequest{
		Protocol:      sofarpc.PROTOCOL_CODE_V1,
		CmdType:       sofarpc.REQUEST,
		CmdCode:       sofarpc.RPC_REQUEST,
		Version:       1,
		ReqID:         uint32(id),
		Codec:         sofarpc.HESSIAN2_SERIALIZE,
		Timeout:       -1,
		RequestHeader: header,
	}
	if len(body) > 0 {
		buf := buffer.NewIoBufferBytes(body)
		cmd.Content = buf
		cmd.ContentLen = buf.Len()
	}
	return cmd, cmd.Content
}

type VerifyConfig struct {
	ExpectedStatus int16
	// if ExepctedHeader is nil, means do not care about header
	// if ExepctedHeader is exists, means resposne Header should contain all the ExpectedHeader
	// TODO :  support regex
	ExpectedHeader map[string]string
	// if ExpectedBody is nil, means do not care about body
	// TODO: support regex
	ExpectedBody []byte
	// if ExpectedRT is zero, means do not care about rt
	// if ExpectedRT is not zero, means response's rt should no more than it
	ExpectedRT time.Duration
}

func (cfg *VerifyConfig) Verify(resp *Response) bool {
	if int16(resp.Status) != cfg.ExpectedStatus {
		fmt.Printf("expected receive status %d, but got %d\n", cfg.ExpectedStatus, resp.Status)
		return false
	}
	if len(resp.Header) < len(cfg.ExpectedHeader) {
		fmt.Printf("expected receive header %v, but got %v\n", cfg.ExpectedHeader, resp.Header)
		return false
	}
	for k, v := range cfg.ExpectedHeader {
		if resp.Header[k] != v {
			fmt.Printf("expected receive header %v, but got %v\n", cfg.ExpectedHeader, resp.Header)
			return false
		}
	}
	if cfg.ExpectedBody != nil {
		if !bytes.Equal(cfg.ExpectedBody, resp.Data) {
			fmt.Printf("expected receive header %s, but got %s\n", string(cfg.ExpectedBody), string(resp.Data))
			return false
		}
	}
	if cfg.ExpectedRT > 0 {
		if resp.Cost > cfg.ExpectedRT {
			fmt.Printf("expected receive rt is %v, but cost %v\n", cfg.ExpectedRT, resp.Cost)
			return false
		}
	}
	return true
}

var DefaultVeirfy = &VerifyConfig{
	ExpectedStatus: sofarpc.RESPONSE_STATUS_SUCCESS,
}
