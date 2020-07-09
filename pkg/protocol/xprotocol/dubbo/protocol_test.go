package dubbo

import (
	"bufio"
	"context"
	"fmt"
	"testing"

	hessian "github.com/apache/dubbo-go-hessian2"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

func Test_dubboProtocol_Hijack(t *testing.T) {
	type args struct {
		request    xprotocol.XFrame
		statusCode uint32
	}
	tests := []struct {
		name  string
		proto *dubboProtocol
		args  args
	}{
		{
			name:  "normal",
			proto: &dubboProtocol{},
			args: struct {
				request    xprotocol.XFrame
				statusCode uint32
			}{
				request: &Frame{
					Header: Header{
						Id: 1,
					},
				},
				statusCode: uint32(types.NoHealthUpstreamCode),
			},
		},
		{
			name:  "status not registry",
			proto: &dubboProtocol{},
			args: struct {
				request    xprotocol.XFrame
				statusCode uint32
			}{
				request: &Frame{
					Header: Header{
						Id: 1,
					},
				},
				statusCode: uint32(99),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := &dubboProtocol{}
			respFrame := proto.Hijack(tt.args.request, tt.args.statusCode)

			buffer, err := encodeFrame(context.TODO(), respFrame.(*Frame))
			if err != nil {
				t.Errorf("%s-> test hijack encode fail:%s", tt.name, err)
				return
			}

			codecR := hessian.NewHessianCodec(bufio.NewReader(buffer))

			h := &hessian.DubboHeader{}
			if err = codecR.ReadHeader(h); err != nil {
				t.Errorf("%s-> hessian encode header fail:%s", tt.name, err)
			}
			if uint64(h.ID) != tt.args.request.GetRequestId() {
				t.Errorf("%s-> dubbo hijack response requestID not equal, want:%d, get:%d", tt.name, tt.args.request.GetRequestId(), h.ID)
				return
			}

			resp := &hessian.Response{}
			if err = codecR.ReadBody(resp); err != nil {
				t.Errorf("%s-> hessian encode body fail:%s", tt.name, err)
				return
			}

			expect, ok := dubboMosnStatusMap[int(tt.args.statusCode)]
			if !ok {
				expect = dubboStatusInfo{
					Status: hessian.Response_SERVICE_ERROR,
					Msg:    fmt.Sprintf("%d status not define", tt.args.statusCode),
				}
			}
			expect.Msg = fmt.Sprintf("java exception:%s", expect.Msg)

			if h.ResponseStatus != expect.Status || resp.Exception.Error() != expect.Msg {
				t.Errorf("%s-> dubbo hijack response or exception fail, input{responseStatus:%d, msg:%s}, output:{responseStatus:%d, msg:%s}", tt.name, tt.args.statusCode, expect.Msg, h.ResponseStatus, resp.Exception.Error())
			}
		})
	}
}
