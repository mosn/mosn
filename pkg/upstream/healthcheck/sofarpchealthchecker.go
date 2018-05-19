package healthcheck

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"
	"math/rand"
	"strconv"

	"reflect"
	"time"
)

type sofarpcHealthChecker struct {
	healthChecker
	//TODO set 'protocolCode' after service subscribe finished
	protocolCode sofarpc.ProtocolType
	serviceName  string
}

// Use for hearth beat starting for sofa bolt in the same codecClient
// for bolt heartbeat, timeout: 90s interval: 15s
func StartSofaHeartBeat(timeout time.Duration, interval time.Duration, hostAddr string,
	codecClient stream.CodecClient, nameHB string, pro sofarpc.ProtocolType) types.HealthCheckSession {

	hcV2 := v2.HealthCheck{
		Timeout:     timeout,
		Interval:    interval,
		ServiceName: nameHB,
	}

	hostV2 := v2.Host{
		Address: hostAddr,
	}

	host := cluster.NewHost(hostV2, nil)

	hc := NewSofarpcHealthCheck(hcV2, pro)
	hcs := hc.newSession(codecClient, host)
	hcs.Start()

	return hcs
}

// Use for hearth beat stopping for sofa bolt in the same codecClient
func StopSofaHeartBeat(hsc types.HealthCheckSession) {
	hsc.Stop()
}

func NewSofarpcHealthCheck(config v2.HealthCheck, pro sofarpc.ProtocolType) *sofarpcHealthChecker {
	hc := NewHealthCheck(config)

	shc := &sofarpcHealthChecker{
		healthChecker: *hc,
		protocolCode:  pro,
	}

	if config.ServiceName != "" {
		shc.serviceName = config.ServiceName
	}

	return shc
}

func (c *sofarpcHealthChecker) newSession(codecClinet stream.CodecClient, host types.Host) types.HealthCheckSession {
	shcs := &sofarpcHealthCheckSession{
		client:             codecClinet,
		healthChecker:      c,
		healthCheckSession: *NewHealthCheckSession(&c.healthChecker, host),
	}

	shcs.intervalTicker = newTicker(shcs.onInterval)
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
	log.DefaultLogger.Debugf("[BoltHealthCheck] get heartbeat message")
	if statusStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)]; ok {
		s.responseStatus = sofarpc.ConvertPropertyValue(statusStr, reflect.Int16).(int16)
	} else if protocolStr, ok := headers[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)]; ok {
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
	// start ticker for periodically sending hb
	s.healthCheckSession.tickerStart()
	s.onInterval()
}

func (s *sofarpcHealthCheckSession) onInterval() {
	if s.client == nil {
		log.DefaultLogger.Debugf("For health check, no codecClient can used")
		connData := s.host.CreateConnection(nil)
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
		log.DefaultLogger.Debugf("[BoltHealthCheck]Sending Heart Beat,request id is %s ...", reqID)
		s.requestEncoder = nil
		s.healthCheckSession.onInterval()
	} else {
		log.DefaultLogger.Fatalf("For health check, only support bolt v1 currently")
	}
}

func (s *sofarpcHealthCheckSession) onTimeout() {
	s.expectReset = true
	s.client.Close()
	s.client = nil

	log.DefaultLogger.Infof("Pay attention: heartbeat client close connection\n")
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
