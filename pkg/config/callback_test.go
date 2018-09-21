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
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/json-iterator/go"
)

type fields struct {
	Servers             []ServerConfig
	ClusterManager      ClusterManagerConfig
	ServiceRegistry     v2.ServiceRegistryInfo
	RawDynamicResources jsoniter.RawMessage
	RawStaticResources  jsoniter.RawMessage
}

// todo fill the unit test
func TestMOSNConfig_OnUpdateEndpoints(t *testing.T) {

	type args struct {
		loadAssignments []*pb.ClusterLoadAssignment
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantError bool
	}{
		{
			name: "Invalid Update",
			fields: fields{
				Servers: []ServerConfig{
					{
						ServerName: "testUpdate",
					},
				},
			},
			args: args{
				loadAssignments: []*pb.ClusterLoadAssignment{
					{
						ClusterName: "test",
					},
				},
			},
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MOSNConfig{
				Servers:             tt.fields.Servers,
				ClusterManager:      tt.fields.ClusterManager,
				ServiceRegistry:     tt.fields.ServiceRegistry,
				RawDynamicResources: tt.fields.RawDynamicResources,
				RawStaticResources:  tt.fields.RawStaticResources,
			}
			if err := config.OnUpdateEndpoints(tt.args.loadAssignments); (err == nil) != tt.wantError {
				t.Errorf("TestMOSNConfig_OnUpdateEndpoints error")
			}
		})
	}
}

func TestMOSNConfig_OnDeleteClusters(t *testing.T) {
	type args struct {
		clusters []*pb.Cluster
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MOSNConfig{
				Servers:             tt.fields.Servers,
				ClusterManager:      tt.fields.ClusterManager,
				ServiceRegistry:     tt.fields.ServiceRegistry,
				RawDynamicResources: tt.fields.RawDynamicResources,
				RawStaticResources:  tt.fields.RawStaticResources,
			}
			config.OnDeleteClusters(tt.args.clusters)
		})
	}
}

func TestMOSNConfig_OnUpdateClusters(t *testing.T) {
	type args struct {
		clusters []*pb.Cluster
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MOSNConfig{
				Servers:             tt.fields.Servers,
				ClusterManager:      tt.fields.ClusterManager,
				ServiceRegistry:     tt.fields.ServiceRegistry,
				RawDynamicResources: tt.fields.RawDynamicResources,
				RawStaticResources:  tt.fields.RawStaticResources,
			}
			config.OnUpdateClusters(tt.args.clusters)
		})
	}
}

func TestMOSNConfig_OnDeleteListeners(t *testing.T) {
	type args struct {
		listeners []*pb.Listener
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MOSNConfig{
				Servers:             tt.fields.Servers,
				ClusterManager:      tt.fields.ClusterManager,
				ServiceRegistry:     tt.fields.ServiceRegistry,
				RawDynamicResources: tt.fields.RawDynamicResources,
				RawStaticResources:  tt.fields.RawStaticResources,
			}
			config.OnDeleteListeners(tt.args.listeners)
		})
	}
}

func TestMOSNConfig_OnAddOrUpdateListeners(t *testing.T) {
	type args struct {
		listeners []*pb.Listener
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MOSNConfig{
				Servers:             tt.fields.Servers,
				ClusterManager:      tt.fields.ClusterManager,
				ServiceRegistry:     tt.fields.ServiceRegistry,
				RawDynamicResources: tt.fields.RawDynamicResources,
				RawStaticResources:  tt.fields.RawStaticResources,
			}
			config.OnAddOrUpdateListeners(tt.args.listeners)
		})
	}
}
