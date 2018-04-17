package healthcheck

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strconv"
)

type sofHealthChecker struct {
	healthChecker
	checkPath   string
	serviceName string
}

func NewSofaHealthCheck(config v2.HealthCheck) types.HealthChecker {
	hc := NewHealthCheck(config)
	hhc := &sofHealthChecker{
		healthChecker: *hc,
		checkPath:     config.CheckPath,
	}

	if config.ServiceName != "" {
		hhc.serviceName = config.ServiceName
	}

	return hhc
}

func (c *sofHealthChecker) newSession(host types.Host) types.HealthCheckSession {
	hhcs := &sofaHealthCheckSession{
		healthChecker:      c,
		healthCheckSession: *NewHealthCheckSession(&c.healthChecker, host),
	}

	hhcs.intervalTimer = newTimer(hhcs.onInterval)
	hhcs.timeoutTimer = newTimer(hhcs.onTimeout)

	return hhcs
}

func (s *sofaHealthCheckSession) onInterval() {
	if s.client == nil {
		connData := s.host.CreateConnection()
		s.client = stream.NewBiDirectCodeClient(protocol.SofaRpc, connData.Connection, connData.HostInfo, s)
		s.expectReset = false
	}

	s.requestEncoder = s.client.NewStream("", s)
	s.requestEncoder.GetStream().AddEventListener(s)

	//Use map[string]string as input
	var reqHeaders interface{}
	reqHeaders = map[string]string{
		sofarpc.SofaPropertyHeader("protocol"): strconv.Itoa(int(sofarpc.PROTOCOL_CODE_V1)),
		sofarpc.SofaPropertyHeader("cmdType"):  strconv.Itoa(int(sofarpc.REQUEST)),
		sofarpc.SofaPropertyHeader("cmdCode"):  strconv.Itoa(int(sofarpc.HEARTBEAT)),
	}

	//Use BoltRequestCommand as input
	reqHeaders = sofarpc.BoltRequestCommand{
		Protocol: sofarpc.PROTOCOL_CODE_V1,
		CmdType:  sofarpc.REQUEST,
		CmdCode:  sofarpc.HEARTBEAT,
	}

	body := []byte{0x01, 0x02} //Body

	s.requestEncoder.EncodeHeaders(reqHeaders.(map[string]string), false)
	s.requestEncoder.EncodeData(buffer.NewIoBufferBytes(body), true)

	s.requestEncoder = nil
	s.healthCheckSession.onInterval()
}

// types.StreamDecoder
type sofaHealthCheckSession struct {
	healthCheckSession

	client          stream.CodecClient
	requestEncoder  types.StreamEncoder
	responseHeaders map[string]string
	healthChecker   *sofHealthChecker
	expectReset     bool
}

// // types.StreamDecoder
func (s *sofaHealthCheckSession) OnDecodeHeaders(headers map[string]string, endStream bool) {
	s.responseHeaders = headers

	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofaHealthCheckSession) OnDecodeData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onResponseComplete()
	}
}

func (s *sofaHealthCheckSession) OnDecodeTrailers(trailers map[string]string) {
	s.onResponseComplete()
}

// overload healthCheckSession
func (s *sofaHealthCheckSession) Start() {
	s.onInterval()
}

func (s *sofaHealthCheckSession) onTimeout() {
	s.expectReset = true
	s.client.Close()
	s.client = nil

	s.healthCheckSession.onTimeout()
}

func (s *sofaHealthCheckSession) onResponseComplete() {
	if s.isHealthCheckSucceeded() {
		s.handleSuccess()
	} else {
		s.handleFailure(types.FailureActive)
	}

	s.client.Close()
	s.client = nil
	s.responseHeaders = nil
}

func (s *sofaHealthCheckSession) isHealthCheckSucceeded() bool {
	if status, ok := s.responseHeaders[sofarpc.SofaPropertyHeader("responsestatus")]; ok {
		statusCode, _ := strconv.Atoi(status)

		return statusCode == 0x0000 //success = 0
	}

	return true
}

func (s *sofaHealthCheckSession) OnResetStream(reason types.StreamResetReason) {
	if s.expectReset {
		return
	}

	s.handleFailure(types.FailureNetwork)
}

func (s *sofaHealthCheckSession) OnAboveWriteBufferHighWatermark() {}

func (s *sofaHealthCheckSession) OnBelowWriteBufferLowWatermark() {}

func (s *sofaHealthCheckSession) OnGoAway() {}

//When you receive health check
func (s *sofaHealthCheckSession) NewStream(streamId string, responseEncoder types.StreamEncoder) types.StreamDecoder {

	as := &requestStream{
		responseEncoder,
		false,
	}
	return as
}

type requestStream struct {
	RespEncoder types.StreamEncoder
	HBFlag      bool
}

func (s *requestStream) OnDecodeHeaders(headers map[string]string, endStream bool) {

	//Deal with request headers, for example
	if cmdType, ok := headers[sofarpc.SofaPropertyHeader("cmdType")]; ok {
		v, _ := strconv.Atoi(cmdType)

		if v == int(sofarpc.REQUEST) {
			cmdCode := headers[sofarpc.SofaPropertyHeader("cmdcode")]
			c, _ := strconv.Atoi(cmdCode)

			if c == int(sofarpc.HEARTBEAT) {
				//Build header beat response
				//Use BoltRequestCommand as input
				s.HBFlag = true
				reqHeaders := sofarpc.BoltRequestCommand{
					Protocol: sofarpc.PROTOCOL_CODE_V1,
					CmdType:  sofarpc.REQUEST,
					CmdCode:  sofarpc.HEARTBEAT,
				}
				s.RespEncoder.EncodeHeaders(reqHeaders, false)
			}
		}
	}
}

func (s *requestStream) OnDecodeData(data types.IoBuffer, endStream bool) {

	//CALL OnEncodeData
	if s.HBFlag {
		msg := []byte{0x0000}
		s.RespEncoder.EncodeData(buffer.NewIoBufferBytes(msg), true)
	}

}

func (s *requestStream) OnDecodeTrailers(trailers map[string]string) {
	//CALL OnEncodeTrailers
}
