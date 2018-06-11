package healthchecktest

import (
	"net"
	"testing"
	"time"
	
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/healthcheck"
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
	conn := network.NewClientConnection(nil, remoteAddr, stopChan, log.DefaultLogger)
	codecClient := stream.NewBiDirectCodeClient(nil, protocol.SofaRpc, conn, nil, nil)
	
	err := conn.Connect(true)
	
	if err != nil {
		log.DefaultLogger.Errorf("Connect to confreg server failed. server = %v", confregServer)
	}
	
	log.DefaultLogger.Debugf("Connect to confreg server. server = %v", confregServer)
	
	
	//todo use conn idle detect to start/stop heartbeat
	healthcheck.StartSofaHeartBeat(10*time.Second, 5*time.Second,
		confregServer, codecClient,sofarpc.HealthName,sofarpc.BOLT_V1)
	time.Sleep(3600*time.Second)
}
