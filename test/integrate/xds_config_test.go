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
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/admin"
	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type effectiveConfig struct {
	MOSNConfig interface{}            `json:"mosn_config,omitempty"`
	Listener   map[string]v2.Listener `json:"listener,omitempty"`
	Cluster    map[string]v2.Cluster  `json:"cluster,omitempty"`
}

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

func handleXdsData(mosnConfig *config.MOSNConfig, xdsFiles []string) error {
	for _, fileName := range xdsFiles {
		file := filepath.Join("testdata", fileName)
		msg := &xdsapi.DiscoveryResponse{}

		if data, err := ioutil.ReadFile(file); err == nil {
			proto.Unmarshal(data, msg)
		} else {
			return err
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
			return errors.New(fmt.Sprintf("unkown type: %s", msg.TypeUrl))
		}
	}
	return nil
}

func TestConfigAddAndUpdate(t *testing.T) {
	mosnConfig := config.Load(filepath.Join("testdata", "envoy.json"))
	admin.Reset()
	admin.SetMOSNConfig(mosnConfig)
	Mosn := mosn.NewMosn(mosnConfig)
	Mosn.Start()

	buf, err := admin.Dump()
	if err != nil {
		t.Fatal(err)
	}
	var m effectiveConfig
	json.Unmarshal(buf, &m)

	if m.MOSNConfig == nil {
		t.Fatalf("mosn_config missing")
	}
	if len(m.Listener) > 0 {
		t.Fatalf("should not have listners")
	}
	if len(m.Cluster) > 0 {
		t.Fatalf("should not have clusters")
	}

	xdsFiles := []string{
		"listener1.input",
		"cluster1.input",
		"clusterloadassignment1.input",
	}
	handleXdsData(mosnConfig, xdsFiles)

	buf, err = admin.Dump()
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(buf, &m)

	if m.MOSNConfig == nil {
		t.Fatalf("mosn_config missing")
	}
	if len(m.Listener) != 35 {
		t.Fatalf("should have 35 listeners, but got %d", len(m.Listener))
	}

	if listener, ok := m.Listener["0.0.0.0_9080"]; !ok {
		t.Fatalf("listener[0.0.0.0_9080] is missing")
	} else {
		if listener.Name != "0.0.0.0_9080" || listener.BindToPort || len(listener.FilterChains) != 1 {
			t.Fatalf("error listener[0.0.0.0_9080] config: %v", listener)
		}

		if len(listener.FilterChains[0].Filters) != 2 {
			t.Fatalf("error listener[0.0.0.0_9080] config: %v", listener)
		}

		var filter v2.Filter
		for _, data := range listener.FilterChains[0].Filters {
			if data.Type == "connection_manager" {
				filter = data
			}
		}
		if data, ok := filter.Config["virtual_hosts"]; !ok {
			t.Fatalf("listener[0.0.0.0_9080] missing virtual_hosts")
		} else {
			hosts := data.([]interface{})
			host := hosts[3].(map[string]interface{})
			routers := host["routers"].([]interface{})
			router := routers[0].(map[string]interface{})
			route := router["route"].(map[string]interface{})
			clusterName := route["cluster_name"].(string)

			// 第一次 reviews 没有按照版本和权重来路由（v1,v2,v3 轮训）
			if clusterName != "outbound|9080||reviews.default.svc.cluster.local" {
				t.Fatalf("reviews.default.svc.cluster.local:9080 should route to [outbound|9080||reviews.default.svc.cluster.local], but got %s", clusterName)
			}
		}
	}

	if len(m.Cluster) != 58 {
		t.Fatalf("should have 58 clusters, but got %d", len(m.Cluster))
	}

	if cluster, ok := m.Cluster["outbound|9080||productpage.default.svc.cluster.local"]; !ok {
		t.Fatalf("cluster[outbound|9080||productpage.default.svc.cluster.local] is missing")
	} else {
		if cluster.Name != "outbound|9080||productpage.default.svc.cluster.local" ||
			cluster.LbType != v2.LB_ROUNDROBIN || len(cluster.Hosts) != 1 {
			t.Fatalf("error cluster config: %v", cluster)
		}

		if cluster.Hosts[0].Address != "172.16.1.171:9080" {
			t.Fatalf("error host: %v", cluster.Hosts[0])
		}
	}

	xdsFiles = []string{
		"listener1.input",
		"cluster1.input",
		"clusterloadassignment1.input",
		"listener2.input",
		"clusterloadassignment2.input",
		"cluster2.input",
		"clusterloadassignment3.input",
		"listener3.input",
		"cluster3.input",
		"clusterloadassignment4.input",
		"listener4.input",
		"cluster4.input",
		"clusterloadassignment5.input",
		"listener5.input",
	}
	handleXdsData(mosnConfig, xdsFiles)

	buf, err = admin.Dump()
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(buf, &m)

	if m.MOSNConfig == nil {
		t.Fatalf("mosn_config missing")
	}
	if len(m.Listener) != 35 {
		t.Fatalf("should have 35 listeners, but got %d", len(m.Listener))
	}

	if listener, ok := m.Listener["0.0.0.0_9080"]; !ok {
		t.Fatalf("listener[0.0.0.0_9080] is missing")
	} else {
		if listener.Name != "0.0.0.0_9080" || listener.BindToPort || len(listener.FilterChains) != 1 {
			t.Fatalf("error listener config: %v", listener)
		}

		var filter v2.Filter
		for _, data := range listener.FilterChains[0].Filters {
			if data.Type == "connection_manager" {
				filter = data
			}
		}

		if data, ok := filter.Config["virtual_hosts"]; !ok {
			t.Fatalf("listener[0.0.0.0_9080] missing virtual_hosts, %v", filter)
		} else {
			hosts := data.([]interface{})
			host := hosts[3].(map[string]interface{})
			routers := host["routers"].([]interface{})
			router := routers[0].(map[string]interface{})
			route := router["route"].(map[string]interface{})
			weightedClusters := route["weighted_clusters"].([]interface{})

			if route["cluster_name"].(string) != "" && len(weightedClusters) != 2 {
				t.Fatalf("reviews.default.svc.cluster.local:9080 should route to weighted_clusters")
			}
			cluster1 := weightedClusters[0].(map[string]interface{})["cluster"].(map[string]interface{})
			cluster2 := weightedClusters[1].(map[string]interface{})["cluster"].(map[string]interface{})

			clusterName1 := cluster1["name"].(string)
			clusterName2 := cluster2["name"].(string)

			weight1 := cluster1["weight"].(float64)
			weight2 := cluster2["weight"].(float64)

			// 第二次 review，按照 v1 和 v3 版本各 50% 的权重路由
			if clusterName1 != "outbound|9080|v1|reviews.default.svc.cluster.local" || weight1 != 50 ||
				clusterName2 != "outbound|9080|v3|reviews.default.svc.cluster.local" || weight2 != 50 {
				t.Fatalf("reviews.default.svc.cluster.local:9080 should route to v1(50) & v3(50)")
			}
		}
	}

	if len(m.Cluster) != 58 {
		t.Fatalf("should have 58 clusters, but got %d", len(m.Cluster))
	}

	if cluster, ok := m.Cluster["outbound|9080||productpage.default.svc.cluster.local"]; !ok {
		t.Fatalf("cluster[outbound|9080||productpage.default.svc.cluster.local] is missing")
	} else {
		if cluster.Name != "outbound|9080||productpage.default.svc.cluster.local" ||
			cluster.LbType != v2.LB_ROUNDROBIN || len(cluster.Hosts) != 1 {
			t.Fatalf("error cluster config: %v", cluster)
		}

		if cluster.Hosts[0].Address != "172.16.1.171:9080" {
			t.Fatalf("error host: %v", cluster.Hosts[0])
		}
	}

	Mosn.Close()
	admin.Reset()
}
