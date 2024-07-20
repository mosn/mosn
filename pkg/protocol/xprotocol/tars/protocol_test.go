package tars

import (
	"math"
	"testing"

	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"

	"github.com/stretchr/testify/assert"
)

func Test_proto_GenerateRequestID(t *testing.T) {
	streamID := uint64(math.MaxUint32 - 1)
	streamID2 := uint64(math.MaxInt32 - 1)
	streamID3 := uint64(math.MaxInt32 - 1 + math.MaxUint32)
	type args struct {
		streamID *uint64
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test max uint32",
			args: args{
				streamID: &streamID,
			},
		},
		{
			name: "test uint32 overflow",
			args: args{
				streamID: &streamID,
			},
		},
		{
			name: "test max int32",
			args: args{
				streamID: &streamID2,
			},
		},
		{
			name: "test int32 overflow",
			args: args{
				streamID: &streamID2,
			},
		},
		{
			name: "test int32 overflow",
			args: args{
				streamID: &streamID3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := tarsProtocol{}
			r := Request{
				cmd: &requestf.RequestPacket{},
			}
			got := proto.GenerateRequestID(tt.args.streamID)
			r.SetRequestId(got)
			assert.Equalf(t, got, r.GetRequestId(), "GenerateRequestID(%v)", tt.args.streamID)
		})
	}
}
