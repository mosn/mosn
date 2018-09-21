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

package integrate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/admin"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/gogo/protobuf/proto"
)

func handleListenersResp(msg *xdsapi.DiscoveryResponse) []*xdsapi.Listener {
	listeners := make([]*xdsapi.Listener, 0)
	for _, res := range msg.Resources {
		listener := xdsapi.Listener{}
		listener.Unmarshal(res.GetValue())
		listeners = append(listeners, &listener)
	}
	return listeners
}

func handleEndpointsResp(msg *xdsapi.DiscoveryResponse) []*xdsapi.ClusterLoadAssignment {
	lbAssignments := make([]*xdsapi.ClusterLoadAssignment, 0)
	for _, res := range msg.Resources {
		lbAssignment := xdsapi.ClusterLoadAssignment{}
		lbAssignment.Unmarshal(res.GetValue())
		lbAssignments = append(lbAssignments, &lbAssignment)
	}
	return lbAssignments
}

func handleClustersResp(msg *xdsapi.DiscoveryResponse) []*xdsapi.Cluster {
	clusters := make([]*xdsapi.Cluster, 0)
	for _, res := range msg.Resources {
		cluster := xdsapi.Cluster{}
		cluster.Unmarshal(res.GetValue())
		clusters = append(clusters, &cluster)
	}
	return clusters
}

func TestConfigAddOrUpdate(t *testing.T) {
	type args struct {
		fileNames []string
	}

	tests := []struct {
		name   string
		args   args
		golden string
	}{
		{
			name: "Bookinfo Normal Routing",
			args: args{
				fileNames: []string{
					"listener1",
					"cluster1",
					"clusterloadassignment1",
				},
			},
			golden: "normal_routing",
		},

		{
			name: "Bookinfo Weight Routing",
			args: args{
				fileNames: []string{
					"listener1",
					"cluster1",
					"clusterloadassignment1",
					"listener2",
					"clusterloadassignment2",
					"cluster2",
					"clusterloadassignment3",
					"listener3",
					"cluster3",
					"clusterloadassignment4",
					"listener4",
					"cluster4",
					"clusterloadassignment5",
					"listener5",
				},
			},
			golden: "weight_routing",
		},
	}

	var mux sync.Mutex

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux.Lock()
			admin.Reset()
			defer func() {
				mux.Unlock()
			}()
			mosnConfig := &config.MOSNConfig{
				RawDynamicResources: []byte{0},
				RawStaticResources:  []byte{0},
			}
			Mosn := mosn.NewMosn(mosnConfig)
			Mosn.Start()

			for _, fileName := range tt.args.fileNames {
				s := filepath.Join("testdata", fileName+".input")
				msg := &xdsapi.DiscoveryResponse{}
				if data, err := ioutil.ReadFile(s); err == nil {
					proto.Unmarshal(data, msg)
				} else {
					t.Error(err)
				}

				switch msg.TypeUrl {
				case "type.googleapis.com/envoy.api.v2.Listener":
					listeners := handleListenersResp(msg)
					fmt.Printf("get %d listeners from LDS\n", len(listeners))
					mosnConfig.OnAddOrUpdateListeners(listeners)
				case "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment":
					endpoints := handleEndpointsResp(msg)
					fmt.Printf("get %d endpoints from EDS\n", len(endpoints))
					mosnConfig.OnUpdateEndpoints(endpoints)
				case "type.googleapis.com/envoy.api.v2.Cluster":
					clusters := handleClustersResp(msg)
					fmt.Printf("get %d clusters from CDS\n", len(clusters))
					mosnConfig.OnUpdateClusters(clusters)
				default:
					t.Errorf("unkown type: %s", msg.TypeUrl)
				}
			}

			effectiveConf := admin.GetEffectiveConfig()
			if data, err := json.MarshalIndent(effectiveConf, "", "  "); err == nil {
				actualJSON := string(data)
				gp := filepath.Join("testdata", tt.golden+".golden")
				want, _ := ioutil.ReadFile(gp)
				expectJSON := string(want)
				if strings.Compare(actualJSON, expectJSON) != 0 {
					t.Errorf("expected: %s\nactual: %s\n", expectJSON, actualJSON)
				}
			} else {
				t.Error(err)
			}
			admin.Reset()
			Mosn.Close()
		})
	}
}
