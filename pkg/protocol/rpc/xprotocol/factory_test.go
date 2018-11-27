package xprotocol

import (
	"context"
	"testing"

	networkbuffer "github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func Test_Coder(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.ContextSubProtocol, "rpc-example")
	coder := &Coder{}
	data := networkbuffer.NewIoBufferBytes([]byte{14, 1, 0, 8, 0, 0, 3, 0})
	_, err := coder.Decode(ctx, data)
	if err == nil {
		t.Errorf("Decode should be fail,because of plugin init not run before testing")
	}
	xRpcCmd := &XRpcCmd{
		ctx:    nil,
		codec:  CreateSubProtocolCodec(nil, SubProtocol("rpc-example")),
		data:   []byte{14, 1, 0, 8, 0, 0, 3, 0},
		header: make(map[string]string),
	}
	_, err2 := coder.Encode(ctx, xRpcCmd)
	if err2 != nil {
		t.Errorf("coder encod fail")
	}
}

func Test_XRpcCmd_Basic(t *testing.T) {
	header := make(map[string]string)
	xRpcCmd := &XRpcCmd{
		ctx:    nil,
		codec:  CreateSubProtocolCodec(nil, SubProtocol("rpc-example")),
		data:   []byte{14, 1, 0, 8, 0, 0, 3, 0},
		header: header,
	}
	if xRpcCmd.ProtocolCode() != 0 {
		t.Errorf("protocol code is no use for xprotocol , should be 0")
	}

	header["method"] = "GET"
	xRpcCmd.SetHeader(header)
	if len(xRpcCmd.Header()) != 1 {
		t.Errorf("protocol code is no use for xprotocol , should be 0")
	}
	xRpcCmd.SetData(nil)
	if xRpcCmd.Data() != nil {
		t.Errorf("protocol code is no use for xprotocol , should be 0")
	}
}

// TODO: add codec unit test,is hard to test codec because of plugin init is not run before unit test
