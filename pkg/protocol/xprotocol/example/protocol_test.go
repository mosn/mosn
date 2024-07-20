package example

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_proto_GenerateRequestID(t *testing.T) {
	streamID := uint64(math.MaxUint32 - 1)
	type args struct {
		streamID *uint64
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test uint32 max",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := proto{}
			r := Request{}
			got := proto.GenerateRequestID(tt.args.streamID)
			r.SetRequestId(got)
			assert.Equalf(t, got, r.GetRequestId(), "GenerateRequestID(%v)", tt.args.streamID)
		})
	}
}
