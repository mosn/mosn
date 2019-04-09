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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
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
