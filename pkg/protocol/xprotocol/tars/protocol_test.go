/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
