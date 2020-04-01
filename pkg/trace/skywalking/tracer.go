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

package skywalking

import (
	"encoding/json"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"github.com/pkg/errors"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

var (
	ReporterCfgErr       = errors.New("SkyWalking tracer support only log and gRPC reporter")
	BackendServiceCfgErr = errors.New("SkyWalking tracer must configure the backend_service")
)

func newGO2SkyTracer(config map[string]interface{}) (t *go2sky.Tracer, err error) {
	cfg, err := parseAndVerifySkyTracerConfig(config)
	if err != nil {
		return nil, err
	}

	var r go2sky.Reporter
	if cfg.Reporter == v2.LogReporter {
		r, err = reporter.NewLogReporter()
		if err != nil {
			return nil, err
		}
	} else if cfg.Reporter == v2.GRPCReporter {
		r, err = reporter.NewGRPCReporter(cfg.BackendService)
		if err != nil {
			return nil, err
		}
	}

	t, err = go2sky.NewTracer(cfg.ServiceName, go2sky.WithReporter(r))
	if err != nil {
		return nil, err
	}

	if cfg.WithRegister {
		// Wait for service and service instance register
		log.DefaultLogger.Infof("[SkyWalking] [tracer] wait go2sky.Tracer register ...")
		t.WaitUntilRegister()
		log.DefaultLogger.Infof("[SkyWalking] [tracer] go2sky.Tracer registered")
	}
	return
}

func parseAndVerifySkyTracerConfig(cfg map[string]interface{}) (config v2.SkyWalkingTraceConfig, err error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return config, err
	}
	log.DefaultLogger.Debugf("[SkyWalking] [tracer] tracer config: %v", string(data))

	// set default value
	config.Reporter = v2.LogReporter
	config.ServiceName = v2.DefaultServiceName
	config.WithRegister = true

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	if config.Reporter != v2.LogReporter && config.Reporter != v2.GRPCReporter {
		return config, ReporterCfgErr
	}

	if config.Reporter == v2.GRPCReporter && config.BackendService == "" {
		return config, BackendServiceCfgErr
	}
	return config, nil
}

type SkyTracer interface {
	// injection go2sky.Tracer
	SetGO2SkyTracer(t *go2sky.Tracer)
}
