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
	goplugin "plugin"

	"mosn.io/api"
	"mosn.io/mosn/pkg/admin/store"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/metrics/shm"
	"mosn.io/mosn/pkg/metrics/sink"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	xwasm "mosn.io/mosn/pkg/protocol/xprotocol/wasm"
	"mosn.io/mosn/pkg/server/keeper"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/wasm"
)

// Init Stages Function
func InitDebugServe(c *v2.MOSNConfig) {
	if c.Debug.StartDebug {
		port := 9090 //default use 9090
		if c.Debug.Port != 0 {
			port = c.Debug.Port
		}
		addr := fmt.Sprintf("0.0.0.0:%d", port)
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

func InitializeMetrics(c *v2.MOSNConfig) {
	metrics.FlushMosnMetrics = true
	initializeMetrics(c.Metrics)
}

func initializeMetrics(config v2.MetricsConfig) {
	// init shm zone
	if config.ShmZone != "" && config.ShmSize > 0 {
		shm.InitDefaultMetricsZone(config.ShmZone, int(config.ShmSize), store.GetMosnState() != store.Active_Reconfiguring)
	}

	// set metrics package
	statsMatcher := config.StatsMatcher
	metrics.SetStatsMatcher(statsMatcher.RejectAll, statsMatcher.ExclusionLabels, statsMatcher.ExclusionKeys)
	metrics.SetMetricsFeature(config.FlushMosn, config.LazyFlush)
	// create sinks
	for _, cfg := range config.SinkConfigs {
		_, err := sink.CreateMetricsSink(cfg.Type, cfg.Config)
		// abort
		if err != nil {
			log.StartLogger.Errorf("[mosn] [init metrics] %s. %v metrics sink is turned off", err, cfg.Type)
			return
		}
		log.StartLogger.Infof("[mosn] [init metrics] create metrics sink: %v", cfg.Type)
	}
}

func InitializePidFile(c *v2.MOSNConfig) {
	initializePidFile(c.Pid)
}

func initializePidFile(pid string) {
	keeper.SetPid(pid)
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
				log.StartLogger.Errorf("[mosn] [init codec] init go-plugin codec failed: %+v", err)
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

	// check protocol factory support
	if factory, ok := codec.(api.XProtocolFactory); ok {
		if err := xprotocol.RegisterProtocolFactory(protocolName, factory); err != nil {
			return err
		}
	} else {
		// register protocol
		if err := xprotocol.RegisterProtocol(protocolName, codec.XProtocol()); err != nil {
			return err
		}
	}

	if err := xprotocol.RegisterMapping(protocolName, codec.HTTPMapping()); err != nil {
		return err
	}
	if err := xprotocol.RegisterMatcher(protocolName, codec.ProtocolMatch()); err != nil {
		return err
	}

	return nil
}
