package healthcheck

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/stream"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"net/http"
	"strconv"
	"strings"
)

type httpHealthChecker struct {
	healthChecker
	checkPath   string
	serviceName string
}

func NewHttpHealthCheck(config v2.HealthCheck) types.HealthChecker {
	hc := NewHealthCheck(config)
	hhc := &httpHealthChecker{
		healthChecker: *hc,
		checkPath:     config.CheckPath,
	}

	if config.ServiceName != "" {
		hhc.serviceName = config.ServiceName
	}

	return hhc
}

func (c *httpHealthChecker) newSession(host types.Host) types.HealthCheckSession {
	hhcs := &httpHealthCheckSession{
		healthChecker:      c,
		healthCheckSession: *NewHealthCheckSession(&c.healthChecker, host),
	}

	hhcs.intervalTimer = newTimer(hhcs.onInterval)
	hhcs.timeoutTimer = newTimer(hhcs.onTimeout)

	return hhcs
}

func (c *httpHealthChecker) createCodecClient(data types.CreateConnectionData) stream.CodecClient {
	return stream.NewCodecClient(protocol.Http2, data.Connection, data.HostInfo)
}

// types.StreamDecoder
type httpHealthCheckSession struct {
	healthCheckSession

	client          stream.CodecClient
	requestEncoder  types.StreamEncoder
	responseHeaders map[string]string
	healthChecker   *httpHealthChecker
	expectReset     bool
}

// // types.StreamDecoder
func (s *httpHealthCheckSession) OnDecodeHeaders(headers map[string]string, endStream bool) {
	s.responseHeaders = headers

	if endStream {
		s.onResponseComplete()
	}
}

func (s *httpHealthCheckSession) OnDecodeData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onResponseComplete()
	}
}

func (s *httpHealthCheckSession) OnDecodeTrailers(trailers map[string]string) {
	s.onResponseComplete()
}

// overload healthCheckSession
func (s *httpHealthCheckSession) Start() {
	s.onInterval()
}

func (s *httpHealthCheckSession) onInterval() {
	if s.client == nil {
		connData := s.host.CreateConnection()
		s.client = s.healthChecker.createCodecClient(connData)
		s.expectReset = false
	}

	s.requestEncoder = s.client.NewStream("", s)
	s.requestEncoder.GetStream().AddEventListener(s)

	reqHeaders := map[string]string{
		types.HeaderMethod: http.MethodGet,
		types.HeaderHost:   s.healthChecker.cluster.Info().Name(),
		types.HeaderPath:   s.healthChecker.checkPath,
	}

	s.requestEncoder.EncodeHeaders(reqHeaders, true)
	s.requestEncoder = nil

	s.healthCheckSession.onInterval()
}

func (s *httpHealthCheckSession) onTimeout() {
	s.expectReset = true
	s.client.Close()
	s.client = nil

	s.healthCheckSession.onTimeout()
}

func (s *httpHealthCheckSession) onResponseComplete() {
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

func (s *httpHealthCheckSession) isHealthCheckSucceeded() bool {
	if status, ok := s.responseHeaders[types.HeaderStatus]; ok {
		statusCode, _ := strconv.Atoi(status)

		return statusCode == 200
	}

	return true
}

func (s *httpHealthCheckSession) OnResetStream(reason types.StreamResetReason) {
	if s.expectReset {
		return
	}

	s.handleFailure(types.FailureNetwork)
}

func (s *httpHealthCheckSession) OnAboveWriteBufferHighWatermark() {}

func (s *httpHealthCheckSession) OnBelowWriteBufferLowWatermark() {}
