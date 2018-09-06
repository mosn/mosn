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

package config

import (
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	xdsendpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
)

// todo fill the unit test
func Test_convertEndpointsConfig(t *testing.T) {
	type args struct {
		xdsEndpoint *xdsendpoint.LocalityLbEndpoints
	}
	tests := []struct {
		name string
		args args
		want []v2.Host
	}{
		{
			name: "case1",
			args: args{
				xdsEndpoint: &xdsendpoint.LocalityLbEndpoints{
					Priority: 1,
				},
			},
			want: []v2.Host{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertEndpointsConfig(tt.args.xdsEndpoint); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertEndpointsConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
