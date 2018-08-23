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
	"math/rand"
	"strconv"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type sofarpcHealthChecker struct {
	healthChecker
	//TODO set 'protocolCode' after service subscribe finished
	protocolCode sofarpc.ProtocolType
}

func newSofaRPCHealthChecker(config v2.HealthCheck) *sofarpcHealthChecker {
	hc := newHealthChecker(config)

	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
	}

	// use bolt v1 as default sofa health check protocol
	if 0 == config.ProtocolCode {
		shc.protocolCode = sofarpc.BOLT_V1
	}

	shc.sessionFactory = shc

	return shc
}

func newSofaRPCHealthCheckerWithBaseHealthChecker(hc *healthChecker, pro sofarpc.ProtocolType) *sofarpcHealthChecker {
	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
		protocolCode:  pro,
	}

	shc.sessionFactory = shc

	return shc
}

func (c *sofarpcHealthChecker) newSofaRPCHealthCheckSession(codecClinet stream.CodecClient, host types.Host) types.HealthCheckSession {
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

func (c *sofarpcHealthChecker) createCodecClient(data types.CreateConnectionData) stream.CodecClient {
	return stream.NewCodecClient(context.Background(), protocol.SofaRPC, data.Connection, data.HostInfo)
}

// types.StreamReceiver
type sofarpcHealthCheckSession struct {
	healthCheckSession

	client         stream.CodecClient
	requestSender  types.StreamSender
	responseStatus int16
	healthChecker  *sofarpcHealthChecker
	expectReset    bool
}

func (s *sofarpcHealthCheckSession) OnReceiveHeaders(headers map[string]string, endStream bool) {
	//bolt
	//log.DefaultLogger.Debugf("BoltHealthCheck get heartbeat message")
	if statusStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)]; ok {
		s.responseStatus = sofarpc.ConvertPropertyValueInt16(statusStr)
	}

	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofarpcHealthCheckSession) OnReceiveData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofarpcHealthCheckSession) OnReceiveTrailers(trailers map[string]string) {
	s.onResponseComplete()
}

func (s *sofarpcHealthCheckSession) OnDecodeError(err error, headers map[string]string) {
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

		s.client = s.healthChecker.createCodecClient(connData)

		s.expectReset = false
	}

	id := rand.Uint32()
	reqID := strconv.Itoa(int(id))

	s.requestSender = s.client.NewStream(context.Background(),reqID, s)
	s.requestSender.GetStream().AddEventListener(s)

	//todo: support tr
	//create protocol specified heartbeat packet
	if s.healthChecker.protocolCode == sofarpc.BOLT_V1 {
		reqHeaders := codec.NewBoltHeartbeat(id)

		s.requestSender.AppendHeaders(reqHeaders, true)
		log.DefaultLogger.Debugf("BoltHealthCheck Sending Heart Beat to %s,request id = %d", s.host.AddressString(), reqID)
		s.requestSender = nil
		// start timeout interval
		s.healthCheckSession.onInterval()
	} else {
		log.DefaultLogger.Errorf("For health check, only support bolt v1 currently")
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

func (s *sofarpcHealthCheckSession) OnResetStream(reason types.StreamResetReason) {
	if s.expectReset {
		return
	}

	s.handleFailure(types.FailureNetwork)
}
