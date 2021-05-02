package tunnel

import (
	"encoding/json"

	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mosn"
	pool "mosn.io/mosn/pkg/stream/connpool/msgconnpool"
	"mosn.io/pkg/utils"
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
	connectionInitFunc := func(c pool.Connection) {
		connData, state := c.GetConnAndState()
		// second check
		if state != pool.Available {
			return
		}
		rawc := connData.Connection.RawConn()
		initInfo := &ConnectionInitInfo{
			ClusterName: config.ClusterName,
			Weight:      config.Weight,
		}
		buffer, err := WriteBuffer(initInfo)
		if err != nil {
			return
		}
		// write connection init request
		rawc.Write(buffer.Bytes())

		// new connection
		utils.GoWithRecover(func() {
			listener.GetListenerCallbacks().OnAccept(rawc, listener.UseOriginalDst(), nil, nil, nil)
		}, nil)
	}
	for i := 0; i < conf.ConnectionNum; i++ {
		conn := pool.NewConn(address, -1, nil, true)
		conn.Init(nil, func() []api.ConnectionEventListener {
			return []api.ConnectionEventListener{&AgentConnectionInitListener{
				initFunc: connectionInitFunc,
				c:        conn,
			}}
		})
	}
}

type ConnectionConfig struct {
	Address     string
	ClusterName string
	Weight      int64
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
