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

package v2

import (
	"reflect"
	"testing"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

func Test_handleRoutesResp(t *testing.T) {
	type args struct {
		resp *envoy_api_v2.DiscoveryResponse
	}
	tests := []struct {
		name string
		args args
		want []*envoy_api_v2.RouteConfiguration
	}{
		{
			name: "case1",
			args: args{
				resp: &envoy_api_v2.DiscoveryResponse{},
			},
			want: []*envoy_api_v2.RouteConfiguration{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client ClientV2
			if got := client.handleRoutesResp(tt.args.resp); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleRoutesResp() = %v, want %v", got, tt.want)
			}
		})
	}
}
