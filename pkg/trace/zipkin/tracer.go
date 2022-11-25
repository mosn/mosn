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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	mosnhttp "mosn.io/mosn/pkg/protocol/http"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

type httpTracer struct {
	serviceName string
	tracer      *zipkin.Tracer
}

func (t *httpTracer) Start(ctx context.Context, request interface{}, startTime time.Time) api.Span {
	header, ok := request.(mosnhttp.RequestHeader)
	if !ok || header.RequestHeader == nil {
		log.DefaultLogger.Debugf("[Zipkin] [tracer] [http1] unable to get request header, downstream trace ignored")
		return NoopSpan
	}
	// weird setup
	localIP, localPort := getLocalHostPort(ctx)
	localEndpoint := &model.Endpoint{
		ServiceName: t.serviceName,
		IPv4:        net.ParseIP(localIP),
		Port:        localPort,
	}
	_ = zipkin.WithLocalEndpoint(localEndpoint)(t.tracer)

	// start span
	spanContext := t.tracer.Extract(extractHTTP(header))
	span := t.tracer.StartSpan(getOperationName(header.RequestURI()),
		zipkin.Parent(spanContext),
		zipkin.Kind(model.Server),
		zipkin.StartTime(startTime),
	)
	return zipkinSpan{
		ztracer: t.tracer,
		zspan:   span,
	}
}

// getLocalHostPort get host and port from context
func getLocalHostPort(ctx context.Context) (string, uint16) {
	v, err := variable.Get(ctx, types.VariableConnection)
	if err == nil {
		if conn, ok := v.(api.Connection); ok {
			host, port, _ := ParseHostPort(conn.LocalAddr().String())
			return host, port
		}
	}
	return "", 0
}

// ParseHostPort parse ip, port from string host
func ParseHostPort(hostPort string) (string, uint16, error) {
	host, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return host, 0, err
	}
	return host, uint16(port), nil
}

func getOperationName(uri []byte) string {
	arr := strings.Split(string(uri), "?")
	return arr[0]
}

func NewHttpTracer(config map[string]interface{}) (api.Tracer, error) {
	cfg, err := parseZipkinConfig(config)
	if err != nil {
		return nil, err
	}
	reporterBuilder, ok := GetReportBuilder(cfg.Reporter)
	if !ok {
		return nil, errors.New(fmt.Sprintf("unsupport report type: %s", cfg.Reporter))
	}
	reporter, err := reporterBuilder(cfg)
	if err != nil {
		log.DefaultLogger.Debugf("[Zipkin] [tracer] [http1] build reporter error: %v", err)
		return nil, err
	}

	sampler, err := zipkin.NewCountingSampler(cfg.SampleRate)
	if err != nil {
		return nil, err
	}

	tracer, err := zipkin.NewTracer(reporter, zipkin.WithSampler(sampler), zipkin.WithTraceID128Bit(true))
	if err != nil {
		return nil, err
	}
	return &httpTracer{
		serviceName: cfg.ServiceName,
		tracer:      tracer,
	}, nil
}

// parseZipkinConfig parse and verify zipkin config
func parseZipkinConfig(config map[string]interface{}) (cfg ZipkinTraceConfig, err error) {
	data, err := json.Marshal(config)
	if err != nil {
		return
	}
	log.DefaultLogger.Debugf("[zipkin] [tracer] tracer config: %v", string(data))

	cfg.ServiceName = v2.DefaultServiceName
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return
	}
	_, err = cfg.ValidateZipkinConfig()
	if err != nil {
		return
	}
	return
}
