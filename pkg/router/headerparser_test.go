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
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func Test_headerParser_evaluateHeaders(t *testing.T) {
	parser := &headerParser{
		headersToAdd: []*headerPair{&headerPair{
			headerName: &lowerCaseString{"level"},
			headerFormatter: &plainHeaderFormatter{
				isAppend:    false,
				staticValue: "1",
			},
		},
		},
		headersToRemove: []*lowerCaseString{&lowerCaseString{"status"}},
	}
	type args struct {
		headers     types.HeaderMap
		requestInfo types.RequestInfo
	}

	tests := []struct {
		name string
		args args
		want types.HeaderMap
	}{
		{
			name: "case1",
			args: args{
				headers:     protocol.CommonHeader{"status": "normal"},
				requestInfo: nil,
			},
			want: protocol.CommonHeader{"level": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.evaluateHeaders(tt.args.headers, tt.args.requestInfo)
			if !reflect.DeepEqual(tt.args.headers, tt.want) {
				t.Errorf("(h *headerParser) evaluateHeaders(headers map[string]string, requestInfo types.RequestInfo) = %v, want %v", tt.args.headers, tt.want)
			}
		})
	}
}
