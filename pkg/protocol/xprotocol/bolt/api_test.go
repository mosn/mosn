package bolt

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/pkg/buffer"
	"testing"
)

// decoded result should equal to request
func TestEncodeDecode(t *testing.T) {
	var (
		ctx            = context.TODO()
		originalHeader = protocol.CommonHeader{
			"k1": "v1",
			"k2": "v2",
		}
		payload   = "hello world"
		requestID = uint32(111)
	)

	//////////// request part start
	req := NewRpcRequest(requestID, originalHeader,
		buffer.NewIoBufferString(payload))

	assert.NotNil(t, req)

	buf, err := encodeRequest(ctx, req)
	assert.Nil(t, err)

	cmd, err := decodeRequest(ctx, buf, false)
	assert.Nil(t, err)

	decodedReq, ok := cmd.(*Request)
	assert.True(t, ok)

	// the decoded header should contains every header set in beginning
	for k, v := range originalHeader {
		val, ok := decodedReq.Get(k)
		assert.True(t, ok)
		assert.Equal(t, v, val)
	}

	// should equal to the original request
	assert.Equal(t, decodedReq.GetData().String(), payload)
	assert.Equal(t, decodedReq.GetRequestId(), uint64(requestID))
	assert.Equal(t, decodedReq.CmdType, CmdTypeRequest)
	assert.False(t, decodedReq.IsHeartbeatFrame())
	assert.Equal(t, decodedReq.GetStreamType(), xprotocol.Request)

	decodedReq.SetRequestId(222)
	assert.Equal(t, decodedReq.GetRequestId(), uint64(222))

	newData := "new hello world"
	decodedReq.SetData(buffer.NewIoBufferString(newData))
	assert.Equal(t, newData, decodedReq.GetData().String())
	//////////// request part end

	//////////// response part start
	resp := NewRpcResponse(requestID, 0, originalHeader, buffer.NewIoBufferString(payload))
	buf, err = encodeResponse(ctx, resp)
	assert.Nil(t, err)

	cmd, err = decodeResponse(ctx, buf)
	assert.Nil(t, err)
	decodedResp, ok := cmd.(*Response)
	assert.True(t, ok)
	assert.Equal(t, decodedResp.GetData().String(), payload)
	assert.Equal(t, decodedResp.GetRequestId(), uint64(requestID))
	assert.Equal(t, decodedResp.CmdType, CmdTypeResponse)
	assert.False(t, decodedResp.IsHeartbeatFrame())
	assert.Equal(t, decodedResp.GetStreamType(), xprotocol.Response)

	newRespData := "new resp data"
	decodedResp.SetData(buffer.NewIoBufferString(newRespData))
	assert.Equal(t, decodedResp.GetData().String(), newRespData)

	decodedResp.SetRequestId(222)
	assert.Equal(t, decodedResp.GetRequestId(), uint64(222))
	//////////// response part end
}
