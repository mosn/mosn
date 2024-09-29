package tunnel

import (
	"encoding/json"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	_ "mosn.io/mosn/pkg/filter/network/connectionmanager"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	"mosn.io/mosn/pkg/filter/network/tunnel"
	"mosn.io/mosn/pkg/filter/network/tunnel/ext"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	_ "mosn.io/mosn/pkg/stream"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/integrate"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

type CustomServerLister struct {
}

var initAddrs = []string{"127.0.0.1:8899", "127.0.0.1:9988"}
var initChan = make(chan struct{})
var addrChangeChan = make(chan struct{})

func (c *CustomServerLister) List(string) chan []string {
	ch := make(chan []string)
	go func() {
		<-initChan
		ch <- initAddrs
		<-addrChangeChan
		ch <- initAddrs[0:1]
	}()
	return ch
}

type CustomConnectionValidator struct {
}

func (c *CustomConnectionValidator) Validate(credential string, host string, cluster string) bool {
	return strings.Contains(credential, cluster)
}

func TestTunnelDynamicServerList(t *testing.T) {
	if os.Getenv("_AGENT_SERVER_") != "" && !strings.Contains(os.Getenv("_AGENT_SERVER_"), "TestTunnelDynamicServerList") {
		return
	}

	util.MeshLogLevel = "DEBUG"
	ext.RegisterServerLister("custom_lister", &CustomServerLister{})
	appaddr := "127.0.0.1:8080"
	agentMeshAddr := "127.0.0.1:2045"
	bootConf := &tunnel.AgentBootstrapConfig{
		Enable:          true,
		ConnectionNum:   1,
		Cluster:         "clientCluster",
		HostingListener: "serverListener",
		DynamicServerListConfig: struct {
			DynamicServerLister string `json:"dynamic_server_lister"`
		}{
			DynamicServerLister: "custom_lister",
		},
	}
	serverMeshAddr1, serviceMeshAddr2 := "127.0.0.1:2046", "127.0.0.1:2047"
	if os.Getenv("_AGENT_SERVER_") == "TestTunnelDynamicServerList1" {
		t.Logf("start1")
		stop := make(chan struct{})
		agentServerConf1 := CreateMeshAgentServer(serverMeshAddr1, initAddrs[0], bolt.ProtocolName)
		agentServer1 := mosn.NewMosn(agentServerConf1)
		agentServer1.Start()
		<-stop
		return
	}

	if os.Getenv("_AGENT_SERVER_") == "TestTunnelDynamicServerList2" {
		t.Logf("start2")
		stop := make(chan struct{})
		agentServerConf2 := CreateMeshAgentServer(serviceMeshAddr2, initAddrs[1], bolt.ProtocolName)
		agentServer2 := mosn.NewMosn(agentServerConf2)
		agentServer2.Start()
		<-stop
		return
	}

	agenConf := CreateMeshWithAgent(agentMeshAddr, appaddr, bootConf, bolt.ProtocolName)
	agentMesh := mosn.NewMosn(agenConf)
	agentMesh.Start()

	pid1 := forkMeshAgentServer("TestTunnelDynamicServerList1")
	if pid1 == 0 {
		t.Fatal("fork error")
		return
	}

	defer syscall.Kill(pid1, syscall.SIGKILL)

	pid2 := forkMeshAgentServer("TestTunnelDynamicServerList2")
	if pid2 == 0 {
		t.Fatal("fork error")
		return
	}

	defer syscall.Kill(pid2, syscall.SIGKILL)

	time.Sleep(time.Second * 10)

	appServer := util.NewRPCServer(t, appaddr, bolt.ProtocolName)
	appServer.GoServe()
	defer appServer.Close()
	// agent server list is empty, will fail
	tc := integrate.NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName))
	go tc.RunCase(1, 0)
	err := <-tc.C
	if err == nil {
		t.Fatal(" error is not expected to be empty")
	}
	t.Logf("try to establiesh agent connection")
	initChan <- struct{}{}
	// Wait for connection is established
	time.Sleep(time.Second * 5)
	tc = integrate.NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName))
	tc.ClientMeshAddr = serviceMeshAddr2
	go tc.RunCase(1, 0)
	err = <-tc.C
	if err != nil {
		t.Fatalf(" error is expected to be empty, %+v", err)
	}

	t.Logf("try to stop one of the servers")
	addrChangeChan <- struct{}{}
	time.Sleep(time.Second * 5)
	go tc.RunCase(1, 0)
	err = <-tc.C
	if err == nil {
		t.Fatalf(" error is not expected to be empty, %+v", err)
	}

	t.Logf("try to send request with serverMeshAddr1")
	tc = integrate.NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName))
	tc.ClientMeshAddr = serverMeshAddr1
	go tc.RunCase(1, 0)
	err = <-tc.C
	if err != nil {
		t.Fatalf(" error is expected to be empty, %+v", err)
	}
	time.Sleep(10 * time.Second)
}

func forkMeshAgentServer(value string) int {
	// Set a flag for the new process start process
	os.Setenv("_AGENT_SERVER_", value)
	log.DefaultLogger.Infof("try to fork %v", value)
	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}

	// Fork exec the new version of your server
	pid, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		return 0
	}
	return pid
}
func CreateMeshAgentServer(meshAddr, tunnelAddr string, subProtocol types.ProtocolName) *v2.MOSNConfig {
	_ = `
{
  "close_graceful": true,
  "servers": [
    {
      "default_log_path": "stdout",
      "default_log_level": "DEBUG",
      "routers":[
        {
          "router_config_name":"client_router",
          "virtual_hosts":[{
            "name":"clientHost",
            "domains": ["*"],
            "routers": [
              {
                "match":{"headers":[{"name":"service","value":".*"}]},
                "route":{"cluster_name":"clientCluster"}
              }
            ]
          }]
        }
      ],
      "listeners": [
        {
          "name":"clientListener",
          "address": "127.0.0.1:2045",
          "bind_port": true,
          "filter_chains": [{
            "filters": [
              {
                "type": "proxy",
                "config": {
                  "downstream_protocol": "X",
                  "upstream_protocol": "X",
                  "extend_config": {
                    "sub_protocol": "bolt"
                  },
                  "router_config_name":"client_router"
                }
              }
            ]
          }]
        },
        {
          "name": "tunnel_server_listener",
          "address": "127.0.0.1:9999",
          "bind_port": true,
          "connection_idle_timeout":0,
          "log_path": "stdout",
          "filter_chains": [
            {
              "tls_context": {},
              "filters": [
                {
                  "type": "tunnel"
                }
              ]
            }
          ]
        }
      ]
    }
  ],
  "cluster_manager": {
    "clusters": [
      {
        "Name": "clientCluster",
        "type": "SIMPLE",
        "lb_type": "LB_RANDOM",
        "max_request_per_conn": 1024,
        "conn_buffer_limit_bytes": 32768
      }
    ]
  }
}
`

	upstreamCluster := "clientCluster"
	downstreamRouters := []v2.Router{
		util.NewHeaderRouter(upstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		util.NewXProtocolFilterChain("proxy", subProtocol, downstreamRouters),
	}
	clientListener := util.NewListener("clientListener", meshAddr, clientChains)

	tunnelChains := []v2.FilterChain{
		{
			FilterChainConfig: v2.FilterChainConfig{
				Filters: []v2.Filter{
					{
						Type: "tunnel",
					},
				},
			},
		},
	}
	tunnelListener := util.NewListener("tunnelListener", tunnelAddr, tunnelChains)

	meshClusterConfig := util.NewBasicCluster(upstreamCluster, nil)

	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
		},
	}
	mosnCfg := util.NewMOSNConfig([]v2.Listener{
		clientListener, tunnelListener,
	}, cmconfig)
	mosnCfg.CloseGraceful = true
	return mosnCfg
}

func CreateMeshWithAgent(meshAddr, backendAddr string, conf *tunnel.AgentBootstrapConfig, subProtocol types.ProtocolName) *v2.MOSNConfig {
	_ = `{
{
  "close_graceful": true,
  "servers": [
    {
      "default_log_path": "stdout",
      "default_log_level": "DEBUG",
      "routers":[
        {
          "router_config_name":"server_router",
          "virtual_hosts":[{
            "name":"serverHost",
            "domains": ["*"],
            "routers": [
              {
                "match":{"headers":[{"name":"service","value":".*"}]},
                "route":{"cluster_name":"serverCluster"}
              }
            ]
          }]
        }
      ],
      "listeners":[
        {
          "name":"serverListener",
          "address": "127.0.0.1:2046",
          "bind_port": true,
          "filter_chains": [{
            "filters": [
              {
                "type": "proxy",
                "config": {
                  "downstream_protocol": "X",
                  "upstream_protocol": "X",
                  "extend_config": {
                    "sub_protocol": "bolt"
                  },
                  "router_config_name":"server_router"
                }
              }
            ]
          }]
        }
      ]
    }
  ],
  "extends": [
    {
      "type": "tunnel_agent",
      "config": {
        "hosting_listener": "serverListener",
        "server_list": ["127.0.0.1:9999"],
        "cluster": "clientCluster",
        "enable": true,
        "connection_num": 1
      }
    }
  ],
  "cluster_manager": {
    "clusters": [
      {
        "Name": "serverCluster",
        "type": "SIMPLE",
        "hosts": [
          {
            "address": "127.0.0.1:8080"
          }
        ]
      }
    ]
  }
}

`

	downstreamCluster := "serverCluster"
	downstreamRouters := []v2.Router{
		util.NewHeaderRouter(downstreamCluster, ".*"),
	}
	clientChains := []v2.FilterChain{
		util.NewXProtocolFilterChain("proxy", subProtocol, downstreamRouters),
	}
	clientListener := util.NewListener("serverListener", meshAddr, clientChains)

	meshClusterConfig := util.NewBasicCluster(downstreamCluster, []string{backendAddr})

	cmconfig := v2.ClusterManagerConfig{
		Clusters: []v2.Cluster{
			meshClusterConfig,
		},
	}

	mosnCfg := util.NewMOSNConfig([]v2.Listener{
		clientListener,
	}, cmconfig)
	confBytes, _ := json.Marshal(conf)

	mosnCfg.Extends = append(mosnCfg.Extends, v2.ExtendConfig{
		Type:   "tunnel_agent",
		Config: confBytes,
	})
	mosnCfg.CloseGraceful = true
	return mosnCfg
}
