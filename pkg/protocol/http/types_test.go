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
package http

import (
	"reflect"
	"testing"

	"mosn.io/mosn/pkg/types"
)

func TestParseQueryString(t *testing.T) {
	type args struct {
		query string
	}
	tests := []struct {
		name string
		args args
		want types.QueryParams
	}{
		{
			args: args{
				query: "",
			},
			want: types.QueryParams{},
		},
		{
			args: args{
				query: "key1=valuex",
			},
			want: types.QueryParams{
				"key1": "valuex",
			},
		},

		{
			args: args{
				query: "key1=valuex&nobody=true",
			},
			want: types.QueryParams{
				"key1":   "valuex",
				"nobody": "true",
			},
		},

		{
			args: args{
				query: "key1=valuex&nobody=true&test=biz",
			},
			want: types.QueryParams{
				"key1":   "valuex",
				"nobody": "true",
				"test":   "biz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseQueryString(tt.args.query); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseQueryString() = %v, want %v", got, tt.want)
			}
		})
	}
}
