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

package healthcheck

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type http2HealthChecker struct {
	healthChecker
	checkPath   string
	serviceName string
}

func newHTTPHealthCheck(config v2.HealthCheck) types.HealthChecker {
	hc := newHealthChecker(config)

	hhc := &http2HealthChecker{
		healthChecker: *hc,
		checkPath:     config.CheckPath,
	}

	hhc.sessionFactory = hc

	return hhc
}

func (c *http2HealthChecker) newSession(host types.Host) types.HealthCheckSession {
	hhcs := &http2HealthCheckSession{
		healthChecker:      c,
		healthCheckSession: *newHealthCheckSession(&c.healthChecker, host),
	}

	hhcs.intervalTimer = newTimer(hhcs.onInterval)
	hhcs.timeoutTimer = newTimer(hhcs.onTimeout)

	return hhcs
}

func (c *http2HealthChecker) createCodecClient(data types.CreateConnectionData) stream.CodecClient {
	return stream.NewCodecClient(context.Background(), protocol.HTTP2, data.Connection, data.HostInfo)
}

// types.StreamReceiver
type http2HealthCheckSession struct {
	healthCheckSession

	client          stream.CodecClient
	requestSender   types.StreamSender
	responseHeaders map[string]string
	healthChecker   *http2HealthChecker
	expectReset     bool
}

// // types.StreamReceiver
func (s *http2HealthCheckSession) OnReceiveHeaders(headers map[string]string, endStream bool) {
	s.responseHeaders = headers

	if endStream {
		s.onResponseComplete()
	}
}

func (s *http2HealthCheckSession) OnReceiveData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onResponseComplete()
	}
}

func (s *http2HealthCheckSession) OnReceiveTrailers(trailers map[string]string) {
	s.onResponseComplete()
}

func (s *http2HealthCheckSession) OnDecodeError(err error, headers map[string]string) {
}

// overload healthCheckSession
func (s *http2HealthCheckSession) Start() {
	s.onInterval()
}

func (s *http2HealthCheckSession) onInterval() {
	if s.client == nil {
		connData := s.host.CreateConnection(nil)
		s.client = s.healthChecker.createCodecClient(connData)
		s.expectReset = false
	}

	s.requestSender = s.client.NewStream(context.Background(),"", s)
	s.requestSender.GetStream().AddEventListener(s)

	reqHeaders := map[string]string{
		types.HeaderMethod:         http.MethodGet,
		types.HeaderHost:           s.healthChecker.cluster.Info().Name(),
		protocol.MosnHeaderPathKey: s.healthChecker.checkPath,
	}

	s.requestSender.AppendHeaders(reqHeaders, true)
	s.requestSender = nil

	s.healthCheckSession.onInterval()
}

func (s *http2HealthCheckSession) onTimeout() {
	s.expectReset = true
	s.client.Close()
	s.client = nil

	s.healthCheckSession.onTimeout()
}

func (s *http2HealthCheckSession) onResponseComplete() {
	if s.isHealthCheckSucceeded() {
		s.handleSuccess()
	} else {
		s.handleFailure(types.FailureActive)
	}

	if conn, ok := s.responseHeaders["connection"]; ok {
		if strings.Compare(strings.ToLower(conn), "close") == 0 {
			s.client.Close()
			s.client = nil
		}
	}

	s.responseHeaders = nil
}

func (s *http2HealthCheckSession) isHealthCheckSucceeded() bool {
	if status, ok := s.responseHeaders[types.HeaderStatus]; ok {
		statusCode, _ := strconv.Atoi(status)

		return statusCode == 200
	}

	return true
}

func (s *http2HealthCheckSession) OnResetStream(reason types.StreamResetReason) {
	if s.expectReset {
		return
	}

	s.handleFailure(types.FailureNetwork)
}
