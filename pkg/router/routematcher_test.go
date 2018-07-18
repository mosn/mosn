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

//
//import (
//	"reflect"
//	"testing"
//
//	"github.com/alipay/sofa-mosn/pkg/api/v2"
//)
//
//func TestNewRouteMatcher(t *testing.T) {
//
//	VHName := "ExampleVH"
//
//	mockProxyAll := &v2.Proxy{
//		VirtualHosts:[]*v2.VirtualHost{
//			{
//				Name:VHName,
//				Domains:[]string{"*","test","*wildcard"},
//				Routers:[]v2.Router{},
//			},
//		},
//	}
//
//	type args struct {
//		config interface{}
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    string
//		wantErr bool
//	}{
//		{
//			name:"testDefault",
//			args:args{
//				config:mockProxyAll,
//			},
//			want:VHName,    // default virtual host's name
//		},
//
//		{
//			name:"testWildcard",
//			args:args{
//				config:mockProxyAll,
//			},
//			want:"wildcard",   // key in tier-2
//		},
//
//		{
//			name:"testVH",
//			args:args{
//				config:mockProxyAll,
//			},
//			want:"test",    // key
//		},
//
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := NewRouteMatcher(tt.args.config)
//
//			rm,_ := got.(*RouteMatcher)
//
//
//			if (err != nil) != tt.wantErr {
//				t.Errorf("NewRouteMatcher() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//
//			if !reflect.DeepEqual(rm.defaultVirtualHost.Name(), tt.want) {
//				t.Errorf("NewRouteMatcher() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
