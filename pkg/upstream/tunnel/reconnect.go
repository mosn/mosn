package tunnel

import (
	"net"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type AgentRawConnection struct {
	ConnectionConfig
	listener types.Listener
}

func NewConnection(config ConnectionConfig, listener types.Listener) *AgentRawConnection {

	if config.Network == "" {
		config.Network = "tcp"
	}
	if config.ReconnectBaseDuration == 0 {
		config.ReconnectBaseDuration = time.Second * 3
	}
	if config.ConnectRetryTimes == 0 {
		config.ConnectRetryTimes = 5
	}
	return &AgentRawConnection{
		ConnectionConfig: config,
		listener:         listener,
	}
}

func (a *AgentRawConnection) connectAndInit() error {
	var rawc net.Conn
	var err error
	backoffConnectDuration := a.ReconnectBaseDuration
	for i := 0; i < int(a.ConnectRetryTimes); i++ {
		rawc, err = net.Dial(a.Network, a.Address)
		if err == nil {
			break
		}
		log.DefaultLogger.Infof("[agent] failed to connect remote server, try again after %v seconds, address: %v, err: %+v", backoffConnectDuration, a.Address, err)
		time.Sleep(backoffConnectDuration)
		backoffConnectDuration *= 2
	}
	if err != nil {
		return err
	}

	initInfo := &ConnectionInitInfo{
		ClusterName: a.ClusterName,
		Weight:      a.Weight,
	}
	buffer, err := WriteBuffer(initInfo)
	if err != nil {
		return nil
	}
	// write connection init request
	_, err = rawc.Write(buffer.Bytes())
	if err != nil {
		// reconnect and write again
		log.DefaultLogger.Infof("[agent] failed to write connection info to remote server, address: %v, err: %+v", a.Address, err)
	}

	// new connection
	utils.GoWithRecover(func() {
		a.listener.GetListenerCallbacks().OnAccept(rawc, a.listener.UseOriginalDst(), nil, nil, nil, []api.ConnectionEventListener{a})
	}, nil)

	return nil
}

func (a *AgentRawConnection) OnEvent(event api.ConnectionEvent) {
	switch {
	case event.IsClose():
		goto RECONNECT
	case event.ConnectFailure():
		goto RECONNECT
	default:
		return
	}

RECONNECT:
	log.DefaultLogger.Infof("[agent] receive reconnect event, and try to reconnect remote server")
	a.connectAndInit()
}
