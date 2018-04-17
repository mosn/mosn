package healthcheck

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"reflect"
)

type sofarpcHealthChecker struct {
	healthChecker
	//TODO set 'protocolCode' after service subscribe finished
	protocolCode byte
	serviceName  string
}

func NewSofarpcHealthCheck(config v2.HealthCheck) types.HealthChecker {
	hc := NewHealthCheck(config)

	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
	}

	if config.ServiceName != "" {
		shc.serviceName = config.ServiceName
	}

	return shc
}

func (c *sofarpcHealthChecker) newSession(host types.Host) types.HealthCheckSession {
	shcs := &sofarpcHealthCheckSession{
		healthChecker:      c,
		healthCheckSession: *NewHealthCheckSession(&c.healthChecker, host),
	}

	shcs.intervalTimer = newTimer(shcs.onInterval)
	shcs.timeoutTimer = newTimer(shcs.onTimeout)

	return shcs
}

func (c *sofarpcHealthChecker) createCodecClient(data types.CreateConnectionData) stream.CodecClient {
	return stream.NewCodecClient(protocol.SofaRpc, data.Connection, data.HostInfo)
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

// // types.StreamDecoder
func (s *sofarpcHealthCheckSession) OnDecodeHeaders(headers map[string]string, endStream bool) {
	//bolt
	if statusStr, ok := headers[sofarpc.SofaPropertyHeader("responsestatus")]; ok {
		s.responseStatus = sofarpc.ConvertPropertyValue(statusStr, reflect.Int16).(int16)
	} else if protocolStr, ok := headers[sofarpc.SofaPropertyHeader("protocol")]; ok {
		//tr protocol, set responseStatus to 'SUCCESS'
		protocol := sofarpc.ConvertPropertyValue(protocolStr, reflect.Uint8).(byte)
		if protocol == sofarpc.PROTOCOL_CODE {
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

// overload healthCheckSession
func (s *sofarpcHealthCheckSession) Start() {
	s.onInterval()
}

func (s *sofarpcHealthCheckSession) onInterval() {
	if s.client == nil {
		connData := s.host.CreateConnection()
		s.client = s.healthChecker.createCodecClient(connData)
		s.expectReset = false
	}

	//TODO random requestId generator
	s.requestEncoder = s.client.NewStream("", s)
	s.requestEncoder.GetStream().AddEventListener(s)

	//create protocol specified heartbeat packet
	//todo: support tr
	reqHeaders := codec.NewBoltHeartbeat(0)

	s.requestEncoder.EncodeHeaders(reqHeaders, true)
	s.requestEncoder = nil

	s.healthCheckSession.onInterval()
}

func (s *sofarpcHealthCheckSession) onTimeout() {
	s.expectReset = true
	s.client.Close()
	s.client = nil

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
