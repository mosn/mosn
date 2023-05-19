//go:build MOSNTest
// +build MOSNTest

package autoconfig

import (
	"strings"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

func TestUpdateCluster(t *testing.T) {
	Scenario(t, "test update mosn cluster config", func() {
		var m *mosn.MosnOperator
		Setup(func() {
			m = mosn.StartMosn(clusterConfig, "-f", "auto_config=true")
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("add new cluster", func() {
			config := `{
				"name": "server_cluster",
				"type": "SIMPLE",
				"lb_type": "LB_RANDOM",
				"hosts":[
					{"address":"127.0.0.1:8080"}
				]
			}`
			err := m.UpdateConfig(34901, "cluster", config)
			Verify(err, Equal, nil)
			content, err := m.GetMosnConfig(34901, "cluster=server_cluster")
			Verify(err, Equal, nil)
			t := strings.Contains(string(content), "127.0.0.1:8080")
			Verify(t, Equal, true)
		})
		Case("update cluster", func() {
			config := `{
				"name": "server_cluster",
				"type": "SIMPLE",
				"lb_type": "LB_ROUNDROBIN",
				"hosts":[
					{"address":"127.0.0.1:8080"},
					{"address":"127.0.0.1:8081"}
				]
			}`
			err := m.UpdateConfig(34901, "cluster", config)
			Verify(err, Equal, nil)
			// wait auto config dump
			time.Sleep(4 * time.Second)
			mcfg := m.LoadMosnConfig() // read config from files
			var c *v2.Cluster
			for _, cc := range mcfg.ClusterManager.Clusters {
				if cc.Name == "server_cluster" {
					c = &cc
				}
			}
			Verify(c, NotNil)
			Verify(c.LbType, Equal, v2.LB_ROUNDROBIN)
			Verify(len(c.Hosts), Equal, 2)
		})
		TearDown(func() {
			m.Stop()
		})
	})
}

const clusterConfig = `{
	"servers":[
		{
			"default_log_path":"stdout"
		}
	],
	"cluster_manager": {},
	"admin": {
		"address": {
			"socket_address": {
				"address": "127.0.0.1",
				"port_value": 34901
			}
		}
	}
}`
