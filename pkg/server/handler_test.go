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

package server

import (
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestDeepEqual(t *testing.T) {

}

// todo add test
func Test_connHandler_AddListener(t *testing.T) {
	type fields struct {
		numConnections int64
		listeners      []*activeListener
		clusterManager types.ClusterManager
		logger         log.Logger
	}
	type args struct {
		lc                     *v2.ListenerConfig
		networkFiltersFactory  types.NetworkFilterChainFactory
		streamFiltersFactories []types.StreamFilterChainFactory
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   types.ListenerEventListener
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &connHandler{
				numConnections: tt.fields.numConnections,
				listeners:      tt.fields.listeners,
				clusterManager: tt.fields.clusterManager,
				logger:         tt.fields.logger,
			}
			if got := ch.AddOrUpdateListener(tt.args.lc, tt.args.networkFiltersFactory, tt.args.streamFiltersFactories); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("connHandler.AddListener() = %v, want %v", got, tt.want)
			}
		})
	}
}
