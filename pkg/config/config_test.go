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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	jsoniter "github.com/json-iterator/go"
)

func TestClusterConfigParse(t *testing.T) {
	mosnConfig := `{
		"cluster_manager": {
			"clusters": [
				{
					"name": "cluster0"
				},
				{
					"name": "cluster1"
				}
			]
		}
	}`
	testConfig := &MOSNConfig{}
	if err := json.Unmarshal([]byte(mosnConfig), testConfig); err != nil {
		t.Fatal(err)
	}
	// verify
	if len(testConfig.ClusterManager.Clusters) != 2 {
		t.Fatal("cluster parsed not enough, got : %v", testConfig.ClusterManager.Clusters)
	}
}

func TestClusterConfigDynamicModeParse(t *testing.T) {
	clusterPath := "/tmp/clusters"
	os.RemoveAll(clusterPath)
	if err := os.MkdirAll(clusterPath, 0755); err != nil {
		t.Fatal(err)
	}
	// dynamic mode
	clusterConfigs := []string{
		`{
			"name": "cluster0"
		}`,
		`{
			"name": "cluster1"
		}`,
	}
	for i, c := range clusterConfigs {
		data := []byte(c)
		fileName := fmt.Sprintf("%s/cluster%d.json", clusterPath, i)
		if err := ioutil.WriteFile(fileName, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// read dynamic mode config
	mosnConfig := `{
		"cluster_manager": {
			"clusters_configs": "/tmp/clusters/"
		}
	}`
	testConfig := &MOSNConfig{}
	if err := json.Unmarshal([]byte(mosnConfig), testConfig); err != nil {
		t.Fatal(err)
	}
	// verify
	if len(testConfig.ClusterManager.Clusters) != 2 {
		t.Fatal("cluster parsed not enough, got : %v", testConfig.ClusterManager.Clusters)
	}
	// add a new cluster
	testConfig.ClusterManager.Clusters = append(testConfig.ClusterManager.Clusters, v2.Cluster{})
	// dump json
	if _, err := json.Marshal(testConfig); err != nil {
		t.Fatal(err)
	}
	// verify
	files, err := ioutil.ReadDir(clusterPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("new cluster is not dumped, just got %d files", len(files))
	}
	// test delete cluster
	testConfig.ClusterManager.Clusters = testConfig.ClusterManager.Clusters[:1]
	// dump json
	if b, err := json.Marshal(testConfig); err != nil || len(b) == 0 {
		t.Fatalf("marshal unexpected, byte: %v, error: %v", b, err)
	}
	files, err = ioutil.ReadDir(clusterPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("new cluster is not dumped, just got %d files", len(files))
	}
}

func TestClusterConfigConflict(t *testing.T) {
	mosnConfig := `{
		"cluster_manager": {
			"clusters_configs": "/tmp/clusters/",
			"clusters": [
				{
					"name": "cluster1"
				}
			]
		}
	}`
	errCompare := func(e error) bool {
		if e == nil {
			return false
		}
		return strings.Contains(e.Error(), v2.ErrDuplicateStaticAndDynamic.Error())
	}
	if err := json.Unmarshal([]byte(mosnConfig), &MOSNConfig{}); !errCompare(err) {
		t.Fatalf("test config conflict with both dynamic mode and static mode failed, get error: %v", err)
	}
}

func TestAdminConfig(t *testing.T) {
	mosnConfig := `{
		"admin": {
			"address": {
				"socket_address": {
					"address": "0.0.0.0",
					"port_value": 34901
				}
			}
		}
	}`
	cfg := &MOSNConfig{}
	if err := json.Unmarshal([]byte(mosnConfig), cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.GetAdmin() == nil {
		t.Error("no admin config got")
	}
}

var _iterJson = jsoniter.ConfigCompatibleWithStandardLibrary

// test for config unmarshal with json-iterator and json (std lib)
func BenchmarkConfigUnmarshal(b *testing.B) {
	// init a config for test
	// assume 5K clusters
	// test dynamic mode
	clusterPath := "/tmp/clusters"
	os.RemoveAll(clusterPath)
	os.MkdirAll(clusterPath, 0755)
	for i := 0; i < 5000; i++ {
		c := fmt.Sprintf(`{
			"name": "cluster%d",
			"type": "SIMPLE",
			"lb_type": "LB_RANDOM"
		}`, i)
		ioutil.WriteFile(fmt.Sprintf("%s/cluster%d.json", clusterPath, i), []byte(c), 0644)
	}
	// benchmark function
	iterBench := func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg := &MOSNConfig{}
			if err := _iterJson.Unmarshal([]byte(cfgStr), cfg); err != nil {
				b.Errorf("json-iterator unmarshal error: %v", err)
				return
			}
		}
	}
	stdBench := func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg := &MOSNConfig{}
			if err := json.Unmarshal([]byte(cfgStr), cfg); err != nil {
				b.Errorf("std json unmarshal error: %v", err)
				return
			}
		}
	}
	b.Run("json-iterator testing", iterBench)
	b.Run("std json testing", stdBench)
}

func BenchmarkConfigMarshal(b *testing.B) {
	clusterPath := "/tmp/clusters"
	cfg := ClusterManagerConfig{
		ClusterManagerConfigJson: ClusterManagerConfigJson{
			ClusterConfigPath: clusterPath,
		},
		Clusters: make([]v2.Cluster, 0, 5000),
	}
	for i := 0; i < 5000; i++ {
		c := v2.Cluster{
			Name:        fmt.Sprintf("cluster%d", i),
			ClusterType: v2.SIMPLE_CLUSTER,
			LbType:      v2.LB_RANDOM,
		}
		cfg.Clusters = append(cfg.Clusters, c)
	}
	conf := &MOSNConfig{
		ClusterManager: cfg,
	}
	// benchmark function
	iterBench := func(b *testing.B) {
		os.RemoveAll(clusterPath)
		os.MkdirAll(clusterPath, 0755)
		for i := 0; i < b.N; i++ {
			if _, err := _iterJson.Marshal(conf); err != nil {
				b.Fatal("json-iterator marshal json error: %v", err)
			}
		}
		// verify
		files, err := ioutil.ReadDir(clusterPath)
		if err != nil {
			b.Fatal("json-iterator verify cluster path failed: %v", err)
		}
		if len(files) != 5000 {
			b.Fatal("json-iterator cluster count is not expected")
		}
	}
	stdBench := func(b *testing.B) {
		os.RemoveAll(clusterPath)
		os.MkdirAll(clusterPath, 0755)

		for i := 0; i < b.N; i++ {
			if _, err := json.Marshal(conf); err != nil {
				b.Fatal("std json marshal json error: %v", err)
			}
		}
		// verify
		files, err := ioutil.ReadDir(clusterPath)
		if err != nil {
			b.Fatal("std json verify cluster path failed: %v", err)
		}
		if len(files) != 5000 {
			b.Fatal("std json cluster count is not expected")
		}
	}
	b.Run("json-iterator testing", iterBench)

	b.Run("std json testing", stdBench)

}
