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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type sofarpcHealthChecker struct {
	healthChecker
	//TODO set 'protocolCode' after service subscribe finished
	protocolCode byte
}

func newSofaRPCHealthChecker(config v2.HealthCheck) *sofarpcHealthChecker {
	hc := newHealthChecker(config)

	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
	}

	// use bolt v1 as default sofa health check protocol
	if 0 == config.ProtocolCode {
		shc.protocolCode = sofarpc.PROTOCOL_CODE_V1
	}

	shc.sessionFactory = shc

	return shc
}

func newSofaRPCHealthCheckerWithBaseHealthChecker(hc *healthChecker, protocolCode byte) *sofarpcHealthChecker {
	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
		protocolCode:  protocolCode,
	}

	shc.sessionFactory = shc

	return shc
}

func (c *sofarpcHealthChecker) newSofaRPCHealthCheckSession(codecClinet stream.Client, host types.Host) types.HealthCheckSession {
	shcs := &sofarpcHealthCheckSession{
		client:             codecClinet,
		healthChecker:      c,
		healthCheckSession: *newHealthCheckSession(&c.healthChecker, host),
	}

	// add timer to trigger hb sending and timeout handling
	shcs.intervalTimer = newTimer(shcs.onInterval)
	shcs.timeoutTimer = newTimer(shcs.onTimeout)

	return shcs
}

func (c *sofarpcHealthChecker) newSession(host types.Host) types.HealthCheckSession {
	shcs := &sofarpcHealthCheckSession{
		healthChecker:      c,
		healthCheckSession: *newHealthCheckSession(&c.healthChecker, host),
	}
	// add timer to trigger hb sending and timeout handling
	shcs.intervalTimer = newTimer(shcs.onInterval)
	shcs.timeoutTimer = newTimer(shcs.onTimeout)

	return shcs
}

func (c *sofarpcHealthChecker) createStreamClient(data types.CreateConnectionData) stream.Client {
	return stream.NewStreamClient(context.Background(), protocol.SofaRPC, data.Connection, data.HostInfo)
}

// types.StreamReceiveListener
type sofarpcHealthCheckSession struct {
	healthCheckSession

	client         stream.Client
	requestSender  types.StreamSender
	responseStatus int16
	healthChecker  *sofarpcHealthChecker
	expectReset    bool
}

func (s *sofarpcHealthCheckSession) OnReceiveHeaders(context context.Context, headers types.HeaderMap, endStream bool) {
	//bolt
	//log.DefaultLogger.Debugf("BoltHealthCheck get heartbeat message")

	switch resp := headers.(type) {
	case rpc.RespStatus:
		s.responseStatus = int16(resp.RespStatus())
	case protocol.CommonHeader:
		if statusStr, ok := headers.Get(sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)); ok {
			s.responseStatus = sofarpc.ConvertPropertyValueInt16(statusStr)
		}
	}

	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofarpcHealthCheckSession) OnReceiveData(context context.Context, data types.IoBuffer, endStream bool) {
	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofarpcHealthCheckSession) OnReceiveTrailers(context context.Context, trailers types.HeaderMap) {
	s.onResponseComplete()
}

func (s *sofarpcHealthCheckSession) OnDecodeError(context context.Context, err error, headers types.HeaderMap) {
}

// overload healthCheckSession
func (s *sofarpcHealthCheckSession) Start() {
	// start interval timer
	s.onInterval()
}

func (s *sofarpcHealthCheckSession) onInterval() {
	if s.client == nil {
		connData := s.host.CreateConnection(nil)

		if err := connData.Connection.Connect(true); err != nil {
			s.handleFailure(types.FailureActive)
			log.DefaultLogger.Debugf("For health check, Connect Error!")
			return
		}

		s.client = s.healthChecker.createStreamClient(connData)

		s.expectReset = false
	}

	s.requestSender = s.client.NewStream(context.Background(), s)
	s.requestSender.GetStream().AddEventListener(s)

	//create protocol specified heartbeat packet
	hbPacket := sofarpc.NewHeartbeat(s.healthChecker.protocolCode)
	if hbPacket != nil {
		s.requestSender.AppendHeaders(context.Background(), hbPacket, true)
		log.DefaultLogger.Debugf("SofaRpc HealthCheck Sending Heart Beat to %s", s.host.AddressString())
		s.requestSender = nil
		// start timeout interval
		s.healthCheckSession.onInterval()
	} else {
		log.DefaultLogger.Errorf("Unknown protocol code: [%x] while sending heartbeat for healthcheck.", s.healthChecker.protocolCode)
	}
}

func (s *sofarpcHealthCheckSession) onTimeout() {
	// todo: fulfil Timeout
	s.expectReset = true
	s.client.Close()
	s.client = nil

	log.DefaultLogger.Errorf("Health Check Timeout for Remote Host = %s", s.host.AddressString())
	// deal with timeout event
	s.healthCheckSession.onTimeout()
}

func (s *sofarpcHealthCheckSession) onResponseComplete() {
	if s.isHealthCheckSucceeded() {
		s.handleSuccess()
	} else {
		s.handleFailure(types.FailureActive)
	}
	//default keepalive in sofarpc, so no need for Connection Header check
	s.responseStatus = -1
}

func (s *sofarpcHealthCheckSession) isHealthCheckSucceeded() bool {
	// -1 or other value considered as failure
	return s.responseStatus == sofarpc.RESPONSE_STATUS_SUCCESS
}

// types.StreamEventListener
func (s *sofarpcHealthCheckSession) OnResetStream(reason types.StreamResetReason) {
	if s.expectReset {
		return
	}

	s.handleFailure(types.FailureNetwork)
}

func (s *sofarpcHealthCheckSession) OnDestroyStream() {}