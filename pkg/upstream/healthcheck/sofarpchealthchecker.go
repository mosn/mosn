package healthcheck

import (
	"math/rand"
	"reflect"
	"strconv"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type sofarpcHealthChecker struct {
	healthChecker
	//TODO set 'protocolCode' after service subscribe finished
	protocolCode sofarpc.ProtocolType
}

func newSofaRpcHealthChecker(config v2.HealthCheck) *sofarpcHealthChecker {
	hc := newHealthChecker(config)

	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
		protocolCode:  sofarpc.ProtocolType(config.ProtocolCode),
	}

	shc.sessionFactory = shc

	return shc
}

func newSofaRpcHealthCheckerWithBaseHealthChecker(hc *healthChecker, pro sofarpc.ProtocolType) *sofarpcHealthChecker {
	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
		protocolCode:  pro,
	}

	shc.sessionFactory = shc

	return shc
}

func (c *sofarpcHealthChecker) newSofaRpcHealthCheckSession(codecClinet stream.CodecClient, host types.Host) types.HealthCheckSession {
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
	return stream.NewCodecClient(nil, protocol.SofaRpc, data.Connection, data.HostInfo)
}

// types.StreamDecoder
type sofarpcHealthCheckSession struct {
	healthCheckSession

	client         stream.CodecClient
	requestEncoder types.StreamEncoder
	responseStatus int16
	healthChecker  *sofarpcHealthChecker
	expectReset    bool
}

func (s *sofarpcHealthCheckSession) OnDecodeHeaders(headers map[string]string, endStream bool) {
	//bolt
	//log.DefaultLogger.Debugf("BoltHealthCheck get heartbeat message")
	if statusStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)]; ok {
		s.responseStatus = sofarpc.ConvertPropertyValue(statusStr, reflect.Int16).(int16)
	} else if protocolStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)]; ok {
		//tr protocol, set responseStatus to 'SUCCESS'
		protocol := sofarpc.ConvertPropertyValue(protocolStr, reflect.Uint8).(byte)
		if protocol == sofarpc.PROTOCOL_CODE_TR {
			s.responseStatus = sofarpc.RESPONSE_STATUS_SUCCESS
		}
	}

	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofarpcHealthCheckSession) OnDecodeData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofarpcHealthCheckSession) OnDecodeTrailers(trailers map[string]string) {
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
		log.DefaultLogger.Debugf("For health check, no codecClient can used")
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

	s.requestEncoder = s.client.NewStream(reqID, s)
	s.requestEncoder.GetStream().AddEventListener(s)

	//todo: support tr
	//create protocol specified heartbeat packet
	if s.healthChecker.protocolCode == sofarpc.BOLT_V1 {
		reqHeaders := codec.NewBoltHeartbeat(id)

		s.requestEncoder.EncodeHeaders(reqHeaders, true)
		log.DefaultLogger.Debugf("BoltHealthCheck Sending Heart Beat to %s,request id = %d", s.host.AddressString(), reqID)
		s.requestEncoder = nil
		// start timeout interval
		s.healthCheckSession.onInterval()
	} else {
		log.DefaultLogger.Errorf("For health check, only support bolt v1 currently")
	}
}

func (s *sofarpcHealthCheckSession) onTimeout() {
	// todo: fulfil Timeout
	s.expectReset = true
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

func (s *sofarpcHealthCheckSession) OnAboveWriteBufferHighWatermark() {}

func (s *sofarpcHealthCheckSession) OnBelowWriteBufferLowWatermark() {}
