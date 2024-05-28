package integrate

import (
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/types"
	_ "mosn.io/mosn/pkg/wasm/runtime/wazero"
	testutil "mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

var WasmProtocol types.ProtocolName = "wasm-bolt" // protocol

type wasmXExtendCase struct {
	*XTestCase
}

func TestWasmProtocolProxy(t *testing.T) {
	appAddr := "127.0.0.1:30999"
	testCases := []*wasmXExtendCase{
		// bolt client -> wasm bolt client -> wasm bolt server -> bolt server
		{NewXTestCase(t, bolt.ProtocolName, testutil.NewRPCServer(t, appAddr, bolt.ProtocolName))},
	}
	for i, tc := range testCases {
		t.Logf("start wasm case #%d\n", i)
		tc.Start(false)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d wasm tcp proxy test failed, protocol: %s, error: %v\n", i, tc.SubProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d wasm tcp proxy hang, protocol: %s\n", i, tc.SubProtocol)
		}
		tc.FinishCase()
	}
}

func (c *wasmXExtendCase) Start(tls bool) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := testutil.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := testutil.CurrentMeshAddr()
	c.ServerMeshAddr = serverMeshAddr

	// proxy to wasm
	cfg := testutil.CreateMeshToMeshConfigWithSub(clientMeshAddr, serverMeshAddr, protocol.Auto, protocol.Auto, WasmProtocol, []string{appAddr}, tls)
	// insert wasm plugin config
	c.injectWasmConfig(cfg)

	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func (c *wasmXExtendCase) injectWasmConfig(cfg *v2.MOSNConfig) {
	cfg.Wasms = []v2.WasmPluginConfig{
		{
			PluginName: "wasm-bolt-plugin",
			VmConfig: &v2.WasmVmConfig{
				Engine: "wasmer",
				Path:   "../../etc/wasm/bolt-go.wasm",
			},
		},
	}
	cfg.ThirdPartCodec = v2.ThirdPartCodecConfig{
		Codecs: []v2.ThirdPartCodec{
			{
				Enable: true,
				Type:   v2.Wasm,
				Config: map[string]interface{}{
					"from_wasm_plugin": "wasm-bolt-plugin",
					"protocol":         string(WasmProtocol),
					"root_id":          1,
				},
			},
		},
	}
}
