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

package router

import (
	"context"
	"reflect"
	"testing"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

func Test_headerParser_evaluateHeaders(t *testing.T) {
	parser := &headerParser{
		headersToAdd: []*headerPair{{
			headerName: "level",
			headerFormatter: &plainHeaderFormatter{
				isAppend:    false,
				staticValue: "1",
			},
		},
		},
		headersToRemove: []string{"status"},
	}
	type args struct {
		ctx     context.Context
		headers types.HeaderMap
	}

	tests := []struct {
		name string
		args args
		want types.HeaderMap
	}{
		{
			name: "case1",
			args: args{
				headers: protocol.CommonHeader{"status": "normal"},
				ctx:     nil,
			},
			want: protocol.CommonHeader{"level": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.evaluateHeaders(tt.args.ctx, tt.args.headers)
			if !reflect.DeepEqual(tt.args.headers, tt.want) {
				t.Errorf("(h *headerParser) evaluateHeaders(ctx context.Context, headers types.HeaderMap) = %v, want %v", tt.args.headers, tt.want)
			}
		})
	}
}
