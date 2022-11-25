package tunnel

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"mosn.io/mosn/pkg/filter/network/tunnel"
	"mosn.io/mosn/pkg/filter/network/tunnel/ext"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/test/integrate"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

func TestTunnelWithCredential(t *testing.T) {
	if os.Getenv("_AGENT_SERVER_") != "" && !strings.Contains(os.Getenv("_AGENT_SERVER_"), "TestTunnelWithCredential") {
		return
	}
	util.MeshLogLevel = "DEBUG"
	appaddr := "127.0.0.1:8081"
	ext.RegisterConnectionCredentialGetter("test_credential", func(cluster string) string {
		return "TestTunnelWithCredential" + cluster
	})
	ext.RegisterConnectionValidator("test_credential", &CustomConnectionValidator{})
	agentMeshAddr := "127.0.0.1:2145"
	tunnelServerListenerAddr := "127.0.0.1:8000"
	bootConf := &tunnel.AgentBootstrapConfig{
		Enable:           true,
		ConnectionNum:    1,
		Cluster:          "clientCluster",
		HostingListener:  "serverListener",
		CredentialPolicy: "test_credential",
		StaticServerList: []string{tunnelServerListenerAddr},
	}
	serverMeshAddr1 := "127.0.0.1:2146"

	if os.Getenv("_AGENT_SERVER_") == "TestTunnelWithCredential1" {
		t.Logf("start1")
		stop := make(chan struct{})
		agentServerConf1 := CreateMeshAgentServer(serverMeshAddr1, tunnelServerListenerAddr, bolt.ProtocolName)
		agentServer1 := mosn.NewMosn(agentServerConf1)
		agentServer1.Start()
		log.DefaultLogger.Infof("start agent server success")
		<-stop
		return
	}
	agenConf := CreateMeshWithAgent(agentMeshAddr, appaddr, bootConf, bolt.ProtocolName)
	agentMesh := mosn.NewMosn(agenConf)
	agentMesh.Start()

	pid1 := forkMeshAgentServer("TestTunnelWithCredential1")
	if pid1 == 0 {
		t.Fatal("fork error")
		return
	}
	defer syscall.Kill(pid1, syscall.SIGKILL)

	time.Sleep(time.Second * 10)

	tc := integrate.NewXTestCase(t, bolt.ProtocolName, util.NewRPCServer(t, appaddr, bolt.ProtocolName))
	tc.ClientMeshAddr = serverMeshAddr1
	tc.AppServer.GoServe()
	defer tc.AppServer.Close()
	go tc.RunCase(1, 0)
	err := <-tc.C
	if err != nil {
		t.Fatal(" error is expected to be empty")
	}
}
