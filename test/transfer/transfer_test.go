package transfer

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/stats"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	"github.com/alipay/sofa-mosn/test/integrate"
	"github.com/alipay/sofa-mosn/test/util"
)

// client - mesh - mesh - server
func forkTransferMesh(tc *integrate.TestCase) int {
	// Set a flag for the new process start process
	os.Setenv("_MOSN_TEST_TRANSFER", "true")

	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}),
	}

	// Fork exec the new version of your server
	pid, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		tc.T.Errorf("Fail to fork %v", err)
		return 0
	}
	return pid
}

func startTransferMesh(tc *integrate.TestCase) {
	server.GracefulTimeout = 5 * time.Second
	network.TransferDomainSocket = "/tmp/mosn.sock"
	stats.TransferDomainSocket = "/tmp/stats.sock"
	cfg := util.CreateMeshToMeshConfig(tc.ClientMeshAddr, tc.ServerMeshAddr, tc.AppProtocol, tc.MeshProtocol, []string{tc.AppServer.Addr()}, true)
	mesh := mosn.NewMosn(cfg)
	/*
		util.MeshLogPath = "./logs/transfer_test.log"
		util.MeshLogLevel = "DEBUG"
		log.InitDefaultLogger(util.MeshLogPath, log.INFO)
		config, _ := json.MarshalIndent(cfg, "", "  ")
		tc.T.Logf("mosn config :%s", string(config))
	*/
	mesh.Start()
	time.Sleep(30 * time.Second)
}

func startTransferServer(tc *integrate.TestCase) {
	tc.AppServer.GoServe()
	go func() {
		<-tc.Stop
		tc.AppServer.Close()
	}()
}

func TestTransfer(t *testing.T) {
	appaddr := "127.0.0.1:8080"

	tc := integrate.NewTestCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(t, appaddr, util.Bolt1))

	tc.ClientMeshAddr = "127.0.0.1:12101"
	tc.ServerMeshAddr = "127.0.0.1:12102"

	if os.Getenv("_MOSN_TEST_TRANSFER") == "true" {
		startTransferMesh(tc)
		return
	}
	pid := forkTransferMesh(tc)
	if pid == 0 {
		t.Fatal("fork error")
		return
	}
	log.InitDefaultLogger("", log.ERROR)
	startTransferServer(tc)

	// wait server and mesh start
	time.Sleep(time.Second)

	// run test cases
	internal := 100 // ms
	for i := 0; i < 10; i++ {
		go tc.RunCase(1000, internal)
	}

	// reload Mosn Server
	time.Sleep(3 * time.Second)
	syscall.Kill(pid, syscall.SIGHUP)

	// test new connection (listen fd transfer)
	time.Sleep(5 * time.Second)
	go tc.RunCase(1000, internal)
	select {
	case err := <-tc.C:
		if err != nil {
			t.Errorf("transfer test failed, error: %v\n", err)
		}
	case <-time.After(20 * time.Second):
	}
	close(tc.Stop)
	time.Sleep(time.Second)
}
