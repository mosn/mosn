package transfer

import (
	"os"
	"syscall"
	"testing"
	"time"

	"math/rand"
	"io/ioutil"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/metrics"
	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/server"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	"github.com/alipay/sofa-mosn/test/integrate"
	"github.com/alipay/sofa-mosn/test/util"
	"encoding/json"
	"github.com/alipay/sofa-mosn/pkg/config"
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
	rand.Seed(3)
	server.GracefulTimeout = 5 * time.Second
	network.TransferDomainSocket = "/tmp/mosn.sock"
	metrics.TransferDomainSocket = "/tmp/stats.sock"
	cfg := util.CreateMeshToMeshConfig(tc.ClientMeshAddr, tc.ServerMeshAddr, tc.AppProtocol, tc.MeshProtocol, []string{tc.AppServer.Addr()}, true)
	cfg.Pid = "/tmp/transfer.pid"

	config.ConfigPath = "/tmp/transfer.json"
	content, err := json.MarshalIndent(cfg, "", "  ")
	if err == nil {
		err = ioutil.WriteFile(config.ConfigPath, content, 0644)
	}

	mesh := mosn.NewMosn(cfg)

	util.MeshLogPath = ""
	util.MeshLogLevel = "DEBUG"
	log.InitDefaultLogger(util.MeshLogPath, log.DEBUG)

	mesh.Start()
	time.Sleep(20 * time.Second)
}

func startTransferServer(tc *integrate.TestCase) {
	tc.AppServer.GoServe()
	go func() {
		<-tc.Finish
		tc.AppServer.Close()
		tc.Finish <- true
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
	log.InitDefaultLogger("", log.DEBUG)
	startTransferServer(tc)

	// wait server and mesh start
	time.Sleep(time.Second)

	// run test cases
	internal := 100 // ms
	// todo: support concurrency
	go tc.RunCase(2000, internal)

	// frist reload Mosn Server, Signal
	time.Sleep(2 * time.Second)
	syscall.Kill(pid, syscall.SIGHUP)

	select {
	case err := <-tc.C:
		if err != nil {
			t.Errorf("transfer test failed, error: %v\n", err)
		}
	case <-time.After(10 * time.Second):
	}

	// second reload Mosn Server, direct start
	forkTransferMesh(tc)

	select {
	case err := <-tc.C:
		if err != nil {
			t.Errorf("transfer test failed, error: %v\n", err)
		}
	case <-time.After(10 * time.Second):
	}
	tc.FinishCase()
}
