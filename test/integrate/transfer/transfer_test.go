package transfer

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"

	"math/rand"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mosn"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/server"
	_ "mosn.io/mosn/pkg/stream/sofarpc"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/integrate"
	"mosn.io/mosn/test/util"
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

func startTransferMesh(t *testing.T, tc *integrate.TestCase) {
	rand.Seed(3)
	server.GracefulTimeout = 5 * time.Second
	types.TransferConnDomainSocket = "/tmp/mosn.sock"
	types.TransferStatsDomainSocket = "/tmp/stats.sock"
	types.TransferListenDomainSocket = "/tmp/listen.sock"
	types.ReconfigureDomainSocket = "/tmp/reconfig.sock"
	// cgo openssl/babassl does not intend to implement tls connection migration for now
	var cfg *v2.MOSNConfig
	if !tls.UseBabasslTag.IsOpen() {
		cfg = util.CreateMeshToMeshConfig(tc.ClientMeshAddr, tc.ServerMeshAddr,
			tc.AppProtocol, tc.MeshProtocol, []string{tc.AppServer.Addr()}, true)
	} else {
		cfg = util.CreateMeshToMeshConfig(tc.ClientMeshAddr, tc.ServerMeshAddr,
			tc.AppProtocol, tc.MeshProtocol, []string{tc.AppServer.Addr()}, false)
	}

	configPath := "/tmp/transfer.json"
	os.Remove(configPath)
	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal("marshal config json failed", err)
	}
	if err := ioutil.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal("write config file failed", err)
	}
	// set config path into load package
	configmanager.Load(configPath)

	mesh := mosn.NewMosn(cfg)

	log.InitDefaultLogger("./transfer.log", log.DEBUG)

	mesh.Start()
	time.Sleep(40 * time.Second)
}

func startTransferServer(tc *integrate.TestCase) {
	tc.AppServer.GoServe()
	tc.DeferFinishCase(func() {
		tc.AppServer.Close()
	})
}

func TestTransfer(t *testing.T) {

	appaddr := "127.0.0.1:8080"

	tc := integrate.NewTestCase(t, protocol.SofaRPC, protocol.SofaRPC, util.NewRPCServer(t, appaddr, util.Bolt1))

	tc.ClientMeshAddr = "127.0.0.1:12101"
	tc.ServerMeshAddr = "127.0.0.1:12102"

	if os.Getenv("_MOSN_TEST_TRANSFER") == "true" {
		startTransferMesh(t, tc)
		return
	}
	pid := forkTransferMesh(tc)
	if pid == 0 {
		t.Fatal("fork error")
		return
	}
	log.InitDefaultLogger("./transfer.log", log.DEBUG)
	startTransferServer(tc)

	// wait server and mesh start
	time.Sleep(time.Second)

	// run test cases
	internal := 100 // ms
	// todo: support concurrency
	go tc.RunCase(5000, internal)

	// frist reload Mosn Server, Signal
	time.Sleep(2 * time.Second)
	syscall.Kill(pid, syscall.SIGHUP)

	select {
	case err := <-tc.C:
		if err != nil {
			t.Errorf("transfer test failed, error: %v\n", err)
		}
	case <-time.After(20 * time.Second):
	}

	// second reload Mosn Server, direct start
	forkTransferMesh(tc)

	select {
	case err := <-tc.C:
		if err != nil {
			t.Errorf("transfer test failed, error: %v\n", err)
		}
	case <-time.After(20 * time.Second):
	}
	tc.FinishCase()
}
