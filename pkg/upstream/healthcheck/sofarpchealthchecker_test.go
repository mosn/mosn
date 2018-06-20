package healthcheck

import (
	"net"
	"testing"
	"time"
	
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
)
const(
	confregServer  string  = "11.166.12.246:9600"
)

func init(){
	log.InitDefaultLogger("",log.DEBUG)
}

func TestStartSofaHeartBeat(t *testing.T) {
	log.DefaultLogger.Debugf("wait 15 seconds")
	remoteAddr, _ := net.ResolveTCPAddr("tcp", confregServer)
	stopChan := make(chan bool,1)
	conn := network.NewClientConnection(nil, nil, remoteAddr, stopChan, log.DefaultLogger)
	codecClient := stream.NewBiDirectCodeClient(nil, protocol.SofaRpc, conn, nil, nil)
	
	err := conn.Connect(true)
	
	if err != nil {
		log.DefaultLogger.Fatalf("Connect to confreg server failed. server = %v", confregServer)
	}
	
	log.DefaultLogger.Infof("Connect to confreg server. server = %v", confregServer)
	
	//todo use conn idle detect to start/stop heartbeat
	StartSofaHeartBeat(config.BoltHeartBeatTimout, config.BoltHeartBeatInterval,
		confregServer, codecClient,config.HealthName,sofarpc.BOLT_V1)
	time.Sleep(3600*time.Second)
}
