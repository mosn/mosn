package server

import (
	"encoding/json"
	"io/ioutil"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/types"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestGetInheritConfig(t *testing.T) {
	tests := []struct {
		name           string
		testConfigPath string
		mosnConfig     string
		wantErr        bool
	}{
		{
			name:           "test Inherit Config",
			testConfigPath: "/tmp/mosn/mosn_admin.json",
			mosnConfig:     mosnConfig,
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.Reset()
			createMosnConfig(tt.testConfigPath, tt.mosnConfig)
			if cfg := configmanager.Load(tt.testConfigPath); cfg != nil {
				store.SetMosnConfig(cfg)
				// init set
				ln := cfg.Servers[0].Listeners[0]
				store.SetListenerConfig(ln.Name, ln)
				cluster := cfg.ClusterManager.Clusters[0]
				store.SetClusterConfig(cluster.Name, cluster)
				router := cfg.Servers[0].Routers[0]
				store.SetRouter(router.RouterConfigName, *router)
			}
			types.InitDefaultPath(configmanager.GetConfigPath())
			dumpConfigBytes, err := store.Dump()
			if err != nil {
				t.Errorf("Dump config error: %v", err)
			}

			effectiveConfigBytes := make([]byte, 0)
			wg := &sync.WaitGroup{}
			wg.Add(2)

			go func() {
				defer wg.Done()
				if err := store.SendInheritConfig(); (err != nil) != tt.wantErr {
					t.Errorf("SendInheritConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
			}()

			go func() {
				defer wg.Done()
				effectiveConfig, err := GetInheritConfig()
				if err != nil {
					t.Errorf("GetInheritConfig error: %v", err)
				}
				effectiveConfigBytes, err = json.Marshal(effectiveConfig)
				if err != nil {
					t.Errorf("json marshal effective Config error: %v", err)
				}
			}()
			wg.Wait()

			if string(dumpConfigBytes) != string(effectiveConfigBytes) {
				t.Errorf("error server.GetInheritConfig:, want: %v, but: %v", string(dumpConfigBytes), string(effectiveConfigBytes))
			}
		})
	}
}

func createMosnConfig(testConfigPath, config string) {

	os.Remove(testConfigPath)
	os.MkdirAll(filepath.Dir(testConfigPath), 0755)

	ioutil.WriteFile(testConfigPath, []byte(config), 0644)

}

const mosnConfig = `{
	"servers": [
                {
                        "mosn_server_name": "test_mosn_server",
                        "listeners": [
                                {
                                         "name": "test_mosn_listener",
                                         "address": "127.0.0.1:8080",
                                         "filter_chains": [
                                                {
                                                        "filters": [
                                                                {
                                                                         "type": "proxy",
                                                                         "config": {
                                                                                 "downstream_protocol": "SofaRpc",
                                                                                 "upstream_protocol": "SofaRpc",
                                                                                 "router_config_name": "test_router"
                                                                         }
                                                                }
                                                        ]
                                                }
                                         ],
                                         "stream_filters": [
                                         ]
                                }
                        ],
                        "routers": [
                                {
                                        "router_config_name": "test_router",
                                        "router_configs": "/tmp/routers/test_router/"
                                }
                        ]
                }
         ],
         "cluster_manager": {
                 "clusters_configs": "/tmp/clusters"
         },
         "admin": {
                 "address": {
                         "socket_address": {
                                 "address": "0.0.0.0",
                                 "port_value": 34901
                         }
                 }
         },
	 "metrics": {
		 "stats_matcher": {
			 "reject_all": true
		 }
	 }

}`
