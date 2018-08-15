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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

func TestGetEnvoyLBMetaData(t *testing.T) {
	header := v2.HeaderMatcher{
		Name:  "service",
		Value: "com.alipay.rpc.common.service.facade.SampleService:1.0",
	}

	var envoyvalue = map[string]interface{}{"label": "gray", "stage": "pre-release"}

	var value = map[string]interface{}{"mosn.lb": envoyvalue}

	routerV2 := v2.Router{
		Match: v2.RouterMatch{
			Headers: []v2.HeaderMatcher{header},
		},

		Route: v2.RouteAction{
			ClusterName: "testclustername",
			MetadataMatch: v2.Metadata{
				"filter_metadata": value,
			},
		},
	}
	type args struct {
		route *v2.Router
	}

	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "testcase1",
			args: args{
				route: &routerV2,
			},
			want: map[string]interface{}{
				"label": "gray", "stage": "pre-release",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMosnLBMetaData(tt.args.route.Route.MetadataMatch); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMosnLBMetaData() = %v, want %v", got, tt.want)
			}
		})
	}
}
