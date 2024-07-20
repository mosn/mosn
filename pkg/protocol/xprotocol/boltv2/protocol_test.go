package boltv2

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_boltProtocol_GenerateRequestID(t *testing.T) {
	streamID := uint64(math.MaxUint32 - 1)
	type args struct {
		streamID *uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "test1",
			args: args{
				streamID: &streamID,
			},
			want: uint64(math.MaxUint32),
		},
		{
			name: "test uint32 overflow",
			args: args{
				streamID: &streamID,
			},
			want: uint64(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := boltv2Protocol{}
			assert.Equalf(t, tt.want, proto.GenerateRequestID(tt.args.streamID), "GenerateRequestID(%v)", tt.args.streamID)
		})
	}
}
