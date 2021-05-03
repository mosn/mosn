package tunnel

import (
	"encoding/json"
	"time"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mosn"
	pool "mosn.io/mosn/pkg/stream/connpool/msgconnpool"
)

type agentBootstrapConfig struct {
	Enable bool `json:"enable"`
	// 建立连接数
	ConnectionNum int `json:"connection_num"`
	// 对应cluster的name
	Cluster string `json:"cluster"`
	// 处理listener name
	HostingListener string `json:"hosting_listener"`
	// Server侧的直连列表
	ServerList []string `json:"server_list"`
}

func init() {
	v2.RegisterParseExtendConfig("agent_bootstrap_config", func(config json.RawMessage) error {
		var conf agentBootstrapConfig
		err := json.Unmarshal(config, &conf)
		if err != nil {
			log.DefaultLogger.Errorf("[tunnel agent] failed to parse agent bootstrap config: %v", err.Error())
			return err
		}
		if conf.Enable {
			bootstrap(&conf)
		}
		return nil
	})
}

func bootstrap(conf *agentBootstrapConfig) {
	for _, serverAddress := range conf.ServerList {
		connectServer(conf, serverAddress)
	}
}

func connectServer(conf *agentBootstrapConfig, address string) {
	servers := mosn.MOSND.GetServer()
	listener := servers[0].Handler().FindListenerByName(conf.HostingListener)
	if listener == nil {
		return
	}
	config := &ConnectionConfig{
		Address:     address,
		ClusterName: conf.Cluster,
		Weight:      10,
	}
	for i := 0; i < conf.ConnectionNum; i++ {
		conn := NewConnection(*config, listener)
		conn.connectAndInit()
	}
}

type ConnectionConfig struct {
	Address               string        `json:"address"`
	ClusterName           string        `json:"cluster_name"`
	Weight                int64         `json:"weight"`
	ConnectRetryTimes     int         `json:"connect_retry_times"`
	Network               string        `json:"network"`
	ReconnectBaseDuration time.Duration `json:"reconnect_base_duration"`
}
type AgentConnectionInitListener struct {
	initFunc func(c pool.Connection)
	c        pool.Connection
}

func (a *AgentConnectionInitListener) OnEvent(event api.ConnectionEvent) {
	if event == api.Connected {
		a.initFunc(a.c)
	}
}
