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

	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestRegisterRouterConfigFactory(t *testing.T) {
	type args struct {
		port    types.Protocol
		factory configFactory
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterRouterConfigFactory(tt.args.port, tt.args.factory)
		})
	}
}

func TestCreateRouteConfig(t *testing.T) {
	type args struct {
		port   types.Protocol
		config interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    types.Routers
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateRouteConfig(tt.args.port, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRouteConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateRouteConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
