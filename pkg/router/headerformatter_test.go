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
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/variable"
	"reflect"
	"testing"
)

func Test_getHeaderFormatter(t *testing.T) {
	_ = variable.Register(variable.NewVariable("address", nil, nil, variable.DefaultSetter, 0))
	type args struct {
		value  string
		append bool
	}
	tests := []struct {
		name string
		args args
		want headerFormatter
	}{
		{
			name: "case1",
			args: args{
				value:  "demo",
				append: false,
			},
			want: &plainHeaderFormatter{
				isAppend:    false,
				staticValue: "demo",
			},
		},
		{
			name: "case2",
			args: args{
				value:  "%address%",
				append: false,
			},
			want: &variableHeaderFormatter{
				isAppend:     false,
				variableName: "address",
			},
		},
		{
			name: "case3",
			args: args{
				value:  "%host%",
				append: false,
			},
			want: nil,
		},
		{
			name: "case4",
			args: args{
				value:  "%address",
				append: false,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHeaderFormatter(tt.args.value, tt.args.append); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHeaderFormatter(value string, append bool) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plainHeaderFormatter_append(t *testing.T) {
	formatter := plainHeaderFormatter{
		isAppend:    false,
		staticValue: "demo",
	}
	tests := []struct {
		name string
		want bool
	}{
		{
			name: "case1",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatter.append(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("(f *plainHeaderFormatter) append() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_plainHeaderFormatter_format(t *testing.T) {
	formatter := plainHeaderFormatter{
		isAppend:    false,
		staticValue: "demo",
	}

	tests := []struct {
		name string
		want string
	}{
		{
			name: "case1",
			want: "demo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatter.format(nil, nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("(f *plainHeaderFormatter) format(types.HeaderMap, types.RequestInfo) = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_variableHeaderFormatter_append(t *testing.T) {
	tests := []struct {
		name      string
		formatter *variableHeaderFormatter
		want      bool
	}{
		{
			name: "case1",
			formatter: &variableHeaderFormatter{
				isAppend:     false,
				variableName: "address",
			},
			want: false,
		},
		{
			name: "case2",
			formatter: &variableHeaderFormatter{
				isAppend:     true,
				variableName: "address",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.formatter.append(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("(v *variableHeaderFormatter) append() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_variableHeaderFormatter_format(t *testing.T) {
	headers := protocol.CommonHeader{"host": "a.com"}
	requestInfo := &network.RequestInfo{}
	requestInfo.SetProtocol("http")

	tests := []struct {
		name      string
		formatter *variableHeaderFormatter
		want      string
	}{
		{
			name: "case1",
			formatter: &variableHeaderFormatter{
				isAppend:     false,
				variableName: "request_header_host",
			},
			want: "a.com",
		},
		{
			name: "case2",
			formatter: &variableHeaderFormatter{
				isAppend:     false,
				variableName: "protocol",
			},
			want: "http",
		},
		{
			name: "case3",
			formatter: &variableHeaderFormatter{
				isAppend:     false,
				variableName: "test",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.formatter.format(headers, requestInfo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("(v *variableHeaderFormatter) format(headers types.HeaderMap, requestInfo api.RequestInfo) = %v, want %v", got, tt.want)
			}
		})
	}
}
