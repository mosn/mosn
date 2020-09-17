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

package zipkin

import (
	"encoding/json"
	"errors"

	"github.com/openzipkin/zipkin-go"
	zipkintracer "github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	zipkinlog "github.com/openzipkin/zipkin-go/reporter/log"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/pkg/log"
)

var (
	ReporterCfgErr       = errors.New("Zipkin tracer support only log and http reporter")
	BackendServiceCfgErr = errors.New("Zipkin tracer must configure the backend_service")
)

func newGO2ZipkinTracer(config map[string]interface{}) (t *zipkin.Tracer, err error) {
	cfg, err := parseAndVerifyZipkinTracerConfig(config)
	if err != nil {
		return nil, err
	}

	switch cfg.Reporter {
	case v2.LogReporter:
		reporter := zipkinlog.NewReporter(nil)
		tracer, err := zipkintracer.NewTracer(reporter,
			zipkintracer.WithSharedSpans(false),
			zipkintracer.WithTraceID128Bit(true),
		)
		if err != nil {
			return nil, err
		}
		return tracer, nil
	case v2.HTTPReporter:
		endpoint, err := zipkintracer.NewEndpoint(cfg.ServiceName, cfg.InstanceIP)
		if err != nil {
			return nil, err
		}

		reporter := zipkinhttp.NewReporter(cfg.BackendURL,
			zipkinhttp.BatchSize(cfg.BatchSize),
		)
		tracer, err := zipkintracer.NewTracer(reporter,
			zipkintracer.WithLocalEndpoint(endpoint),
			zipkintracer.WithSharedSpans(false),
			zipkintracer.WithTraceID128Bit(true),
		)
		if err != nil {
			return nil, err
		}
		return tracer, nil
	default:
		return nil, ReporterCfgErr
	}
}

func parseAndVerifyZipkinTracerConfig(cfg map[string]interface{}) (config v2.ZipkinTraceConfig, err error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return config, err
	}
	log.DefaultLogger.Debugf("[Zipkin] [tracer] tracer config: %v", string(data))

	// set default value
	config.Reporter = v2.LogReporter
	config.ServiceName = v2.DefaultServiceName

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	if config.Reporter != v2.LogReporter && config.Reporter != v2.HTTPReporter {
		return config, ReporterCfgErr
	}

	if config.Reporter == v2.HTTPReporter && config.BackendURL == "" {
		return config, BackendServiceCfgErr
	}
	return config, nil
}

type zipkinTracer interface {
	// injection zipkin.Tracer
	SetGO2ZipkinTracer(t *zipkin.Tracer)
}
