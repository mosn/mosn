/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mosn

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	goplugin "plugin"

	"mosn.io/api"
	"mosn.io/mosn/pkg/admin/store"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	xwasm "mosn.io/mosn/pkg/protocol/xprotocol/wasm"
	"mosn.io/mosn/pkg/server/pid"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm"
)

// Init Stages Function
func InitDebugServe(c *v2.MOSNConfig) {
	if c.Debug.StartDebug {
		port := 9090 //default use 9090
		endpoint := "0.0.0.0"
		if c.Debug.Port != 0 {
			port = c.Debug.Port
		}
		if c.Debug.Endpoint != "" {
			endpoint = c.Debug.Endpoint
		}
		addr := fmt.Sprintf("%s:%d", endpoint, port)
		s := &http.Server{Addr: addr, Handler: nil}
		store.AddService(s, "pprof", nil, nil)
	}
}

// Init Stages Function
func InitializeTracing(c *v2.MOSNConfig) {
	initializeTracing(c.Tracing)
}

func initializeTracing(config v2.TracingConfig) {
	if config.Enable && config.Driver != "" {
		err := trace.Init(config.Driver, config.Config)
		if err != nil {
			log.StartLogger.Errorf("[mosn] [init tracing] init driver '%s' failed: %s, tracing functionality is turned off.", config.Driver, err)
			trace.Disable()
			return
		}
		log.StartLogger.Infof("[mosn] [init tracing] enable tracing")
		trace.Enable()
	} else {
		log.StartLogger.Infof("[mosn] [init tracing] disable tracing")
		trace.Disable()
	}
}

func InitDefaultPath(c *v2.MOSNConfig) {
	types.InitDefaultPath(configmanager.GetConfigPath(), c.UDSDir)
}

func InitializePidFile(c *v2.MOSNConfig) {
	initializePidFile(c.Pid)
}

func initializePidFile(id string) {
	pid.SetPid(id)
}

func InitializePlugin(c *v2.MOSNConfig) {
	initializePlugin(c.Plugin.LogBase)
}

func initializePlugin(log string) {
	if log == "" {
		log = types.MosnLogBasePath
	}
	plugin.InitPlugin(log)
}

func InitializeWasm(c *v2.MOSNConfig) {
	initializeWasm(c.Wasms)
}

func initializeWasm(wasms []v2.WasmPluginConfig) {
	// initialize wasm, shoud before the creation of server listener since the initialization of streamFilter might rely on wasm config
	for _, config := range wasms {
		_ = wasm.GetWasmManager().AddOrUpdateWasm(config)
	}
}

func InitializeThirdPartCodec(c *v2.MOSNConfig) {
	initializeThirdPartCodec(c.ThirdPartCodec)
}

func initializeThirdPartCodec(config v2.ThirdPartCodecConfig) {
	for _, codec := range config.Codecs {
		if !codec.Enable {
			log.StartLogger.Infof("[mosn] [init codec] third part codec disabled for %+v, skip...", codec.Path)
			continue
		}

		switch codec.Type {
		case v2.GoPlugin:
			if err := readProtocolPlugin(codec.Path, codec.LoaderFuncName); err != nil {
				cwd, _ := os.Getwd()
				log.StartLogger.Errorf("[mosn] [init codec] init go-plugin codec failed: %+v, cwd: %v)", err, cwd)
				continue
			}
			log.StartLogger.Infof("[mosn] [init codec] load go plugin codec succeed: %+v", codec.Path)

		case v2.Wasm:
			if err := xwasm.GetProxyProtocolManager().AddOrUpdateProtocolConfig(codec.Config); err != nil {
				log.StartLogger.Errorf("[mosn] [init codec] init wasm codec failed: %+v", err)
				continue
			}
			log.StartLogger.Errorf("[mosn] [init codec] load wasm codec succeed: %+v", codec.Path)
		default:
			log.StartLogger.Errorf("[mosn] [init codec] unknown third part codec type: %+v", codec.Type)
		}
	}
}

const (
	DefaultLoaderFunctionName string = "LoadCodec"
)

func readProtocolPlugin(path, loadFuncName string) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return err
	}

	if loadFuncName == "" {
		loadFuncName = DefaultLoaderFunctionName
	}

	sym, err := p.Lookup(loadFuncName)
	if err != nil {
		return err
	}

	loadFunc := sym.(func() api.XProtocolCodec)
	codec := loadFunc()

	protocolName := codec.ProtocolName()
	log.StartLogger.Infof("[mosn] [init codec] loading protocol [%v] from third part codec", protocolName)

	if err := xprotocol.RegisterXProtocolCodec(codec); err != nil {
		return err
	}

	return nil
}
