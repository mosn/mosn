package bolt

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"net/http"
	"testing"
)

func TestProto(t *testing.T) {
	var (
		bp      = boltProtocol{}
		ctx     = context.TODO()
		payload = "hello world"
		header  = protocol.CommonHeader{"k": "v"}
		req     = NewRpcRequest(111, header, buffer.NewIoBufferString(payload))
		resp    = NewRpcResponse(111, 0, header, buffer.NewIoBufferString(payload))
	)

	assert.Equal(t, bp.Name(), ProtocolName)

	/////// request
	buf, err := bp.Encode(ctx, req)
	assert.Nil(t, err)

	cmdInter, err := bp.Decode(ctx, buf)
	assert.Nil(t, err)
	cmd, ok := cmdInter.(*Request)
	assert.True(t, ok)
	assert.Equal(t, cmd.Content.String(), payload)
	/////// request end

	/////// heartbeat
	frame := bp.Trigger(111)
	assert.NotNil(t, frame)
	assert.Equal(t, frame.(*Request).RequestHeader.CmdType, CmdTypeRequest)
	assert.Equal(t, frame.(*Request).RequestHeader.CmdCode, CmdCodeHeartbeat)
	/////// heartbeat end

	/////// response
	buf, err = bp.Encode(ctx, resp)
	assert.Nil(t, err)

	cmdInter, err = bp.Decode(ctx, buf)
	assert.Nil(t, err)
	cmdResp, ok := cmdInter.(*Response)
	assert.True(t, ok)
	assert.Equal(t, cmdResp.Content.String(), payload)
	/////// response end
}

func TestMapping(t *testing.T) {
	var (
		bp      = boltProtocol{}
		mapping = map[uint32]uint32{
			http.StatusOK:               uint32(ResponseStatusSuccess),
			types.RouterUnavailableCode: uint32(ResponseStatusNoProcessor),
			types.NoHealthUpstreamCode:  uint32(ResponseStatusConnectionClosed),
			types.UpstreamOverFlowCode:  uint32(ResponseStatusServerThreadpoolBusy),
			types.CodecExceptionCode:    uint32(ResponseStatusCodecException),
			types.DeserialExceptionCode: uint32(ResponseStatusServerDeserialException),
			types.TimeoutExceptionCode:  uint32(ResponseStatusTimeout),
			999999:                      uint32(ResponseStatusUnknown),
		}
	)

	for k, v := range mapping {
		assert.Equal(t, bp.Mapping(k), v)
	}
}

func TestReply(t *testing.T) {
	bp := boltProtocol{}
	// reply heartbeat
	resp := bp.Reply(NewRpcResponse(1, 0, nil, buffer.NewIoBufferString("hello")))
	assert.True(t, resp.IsHeartbeatFrame())
}

func TestHijack(t *testing.T) {
	bp := boltProtocol{}
	rsp := NewRpcResponse(1, 0, nil, buffer.NewIoBufferString("hello"))
	frame := bp.Hijack(rsp, 999)
	assert.Equal(t, frame.GetStatusCode(), uint32(999))
}

func TestBufferReset(t *testing.T) {
	req := NewRpcRequest(111, nil, buffer.NewIoBufferString("hell"))
	rsp := NewRpcResponse(1, 0, nil, buffer.NewIoBufferString("hello"))
	var buf = &boltBuffer{*req, *rsp}
	ins.Reset(buf)

	assert.Nil(t, buf.request.Content)
	assert.Nil(t, buf.response.Content)
}
