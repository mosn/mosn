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
