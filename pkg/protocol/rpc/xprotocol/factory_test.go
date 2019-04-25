package xprotocol

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"testing"

	networkbuffer "github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"

	mosnctx "github.com/alipay/sofa-mosn/pkg/context"
)

func init() {
	Register("ut-example", &rpcTestFactory{})
}

type rpcTestFactory struct{}

func (ref *rpcTestFactory) CreateSubProtocolCodec(context context.Context) Multiplexing {
	return NewTestExample()
}

type testExample struct{}

// NewRPCExample create rpc-example codec
func NewTestExample() Multiplexing {
	return &testExample{}
}

var (
	// ReqIDLen rpc-example req id len
	ReqIDLen = 8
	// ReqIDBeginOffset rpc-example req id field offset
	ReqIDBeginOffset = 0
	// ReqDataLen rpc-example request package size
	ReqDataLen = 8
)

func (re *testExample) SplitFrame(data []byte) [][]byte {
	var reqs [][]byte
	start := 0
	dataLen := len(data)
	for true {
		if dataLen >= ReqDataLen {
			// there is one valid rpc-example request
			reqs = append(reqs, data[start:(start+ReqDataLen)])
			start += ReqDataLen
			dataLen -= ReqDataLen
			if dataLen == 0 {
				break
			}
		} else {
			break
		}
	}
	return reqs
}

func (re *testExample) GetStreamID(data []byte) string {
	reqIDRaw := data[ReqIDBeginOffset:ReqIDLen]
	reqID := binary.BigEndian.Uint64(reqIDRaw)
	reqIDStr := fmt.Sprintf("%d", reqID)
	return reqIDStr
}

func (re *testExample) SetStreamID(data []byte, streamID string) []byte {
	reqID, err := strconv.ParseInt(streamID, 10, 64)
	if err != nil {
		return data
	}
	buf := bytes.Buffer{}
	err = binary.Write(&buf, binary.BigEndian, reqID)
	if err != nil {
		return data
	}
	reqIDStr := buf.Bytes()
	reqIDStrLen := len(reqIDStr)
	for i := 0; i < ReqIDLen && i <= reqIDStrLen; i++ {
		data[ReqIDBeginOffset+i] = reqIDStr[i]
	}
	return data
}

//Tracing
func (re *testExample) GetServiceName(data []byte) string {
	return "test-service-name"
}
func (re *testExample) GetMethodName(data []byte) string {
	return "test-method-name"
}

//RequestRouting
func (re *testExample) GetMetas(data []byte) map[string]string {
	metas := make(map[string]string)
	metas["ua"] = "firefox"
	return metas
}

//ProtocolConvertor
func (re *testExample) Convert(data []byte) (map[string]string, []byte) {
	return nil, nil
}

func Test_Engine_Coder(t *testing.T) {
	ctx := mosnctx.WithValue(context.Background(), types.ContextSubProtocol, "ut-example")
	engine := Engine()
	data := networkbuffer.NewIoBufferBytes([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	xRpcCmd, err := engine.Decode(ctx, data)
	if err != nil {
		t.Errorf("xprotocol engine decode data fail")
	}
	_, err2 := engine.Encode(ctx, xRpcCmd)
	if err2 != nil {
		t.Errorf("coder encod fail")
	}
}

func Test_XRpcCmd_0(t *testing.T) {
	header := make(map[string]string)
	xRpcCmd := &XRpcCmd{
		ctx:    nil,
		codec:  CreateSubProtocolCodec(nil, SubProtocol("ut-example")),
		data:   networkbuffer.NewIoBufferBytes([]byte{14, 1, 0, 8, 0, 0, 3, 0}),
		header: header,
	}

	// RpcCmd test
	if xRpcCmd.ProtocolCode() != 0 {
		t.Errorf("protocol code is no use for xprotocol , should be 0")
	}
	header["method"] = "GET"
	xRpcCmd.SetHeader(header)
	if len(xRpcCmd.Header()) != 1 {
		t.Errorf("header len should be 1,set header fail")
	}
	xRpcCmd.SetData(nil)
	if xRpcCmd.Data() != nil {
		t.Errorf("data should be nil, set data fail")
	}

	xRpcCmd.SetData(networkbuffer.NewIoBufferBytes([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	xRpcCmd.SetRequestID(1)
	if xRpcCmd.RequestID() != 1 {
		t.Errorf("set request id fail,should be 1")
	}
}

func Test_XRpcCmd_1(t *testing.T) {
	header := make(map[string]string)
	xRpcCmd := &XRpcCmd{
		ctx:    nil,
		codec:  CreateSubProtocolCodec(nil, SubProtocol("ut-example")),
		data:   networkbuffer.NewIoBufferBytes([]byte{14, 1, 0, 8, 0, 0, 3, 0}),
		header: header,
	}
	// Xprotocol plugin test
	frames := xRpcCmd.SplitFrame([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	if len(frames) != 1 {
		t.Errorf("split frame fail")
	}
	for _, frame := range frames {
		tData, tHeader := xRpcCmd.Convert(frame)
		if tData != nil || tHeader != nil {
			t.Errorf("convert fail")
		}

		serviceName := xRpcCmd.GetServiceName(frame)
		if serviceName != "test-service-name" {
			t.Errorf("get service name fail")
		}
		methodName := xRpcCmd.GetMethodName(frame)
		if methodName != "test-method-name" {
			t.Errorf("get method name fail")
		}

		metas := xRpcCmd.GetMetas(frame)
		if len(metas) != 1 || metas["ua"] != "firefox" {
			t.Errorf("get metas fail")
		}
	}
}

func Test_XRpcCmd_2(t *testing.T) {
	header := make(map[string]string)
	xRpcCmd := &XRpcCmd{
		ctx:    nil,
		codec:  CreateSubProtocolCodec(nil, SubProtocol("ut-example")),
		data:   networkbuffer.NewIoBufferBytes([]byte{14, 1, 0, 8, 0, 0, 3, 0}),
		header: header,
	}
	// HeaderMap test
	xRpcCmd.Set("ua", "firefox")
	value, ok := xRpcCmd.Get("ua")
	if !ok {
		t.Errorf("set and get header key fail")
	} else {
		if value != "firefox" {
			t.Errorf("set header key fail")
		}
	}
	count := 0
	xRpcCmd.Range(func(key, value string) bool {
		count += 1
		return true
	})
	if count != 1 {
		t.Errorf("test range fail")
	}

	xRpcCmd.Del("ua")
	_, ok2 := xRpcCmd.Get("ua")
	if ok2 {
		t.Error("delete header key fail")
	}

	if xRpcCmd.ByteSize() != 0 {
		t.Errorf("test byte size fail")
	}
}
