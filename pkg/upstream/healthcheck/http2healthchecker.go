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

type http2HealthChecker struct {
	healthChecker
	checkPath   string
	serviceName string
}

func NewHttpHealthCheck(config v2.HealthCheck) types.HealthChecker {
	hc := NHCInstance.NewHealthCheck(config)
	if hcc, ok := hc.(*healthChecker); ok {

		hhc := &http2HealthChecker{
			healthChecker: *hcc,
			checkPath:     config.CheckPath,
		}

		if config.ServiceName != "" {
			hhc.serviceName = config.ServiceName
		}

		return hhc
	}
	return nil
}

func (c *http2HealthChecker) newSession(host types.Host) types.HealthCheckSession {
	hhcs := &http2HealthCheckSession{
		healthChecker:      c,
		healthCheckSession: *NewHealthCheckSession(&c.healthChecker, host),
	}

	hhcs.intervalTimer = newTimer(hhcs.onInterval)
	hhcs.timeoutTimer = newTimer(hhcs.onTimeout)

	return hhcs
}

func (c *http2HealthChecker) createCodecClient(data types.CreateConnectionData) stream.CodecClient {
	return stream.NewCodecClient(nil, protocol.Http2, data.Connection, data.HostInfo)
}

// types.StreamDecoder
type http2HealthCheckSession struct {
	healthCheckSession

	client          stream.CodecClient
	requestEncoder  types.StreamEncoder
	responseHeaders map[string]string
	healthChecker   *http2HealthChecker
	expectReset     bool
}

// // types.StreamDecoder
func (s *http2HealthCheckSession) OnDecodeHeaders(headers map[string]string, endStream bool) {
	s.responseHeaders = headers

	if endStream {
		s.onResponseComplete()
	}
}

func (s *http2HealthCheckSession) OnDecodeData(data types.IoBuffer, endStream bool) {
	if endStream {
		s.onResponseComplete()
	}
}

func (s *http2HealthCheckSession) OnDecodeTrailers(trailers map[string]string) {
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

func (s *http2HealthCheckSession) OnAboveWriteBufferHighWatermark() {}

func (s *http2HealthCheckSession) OnBelowWriteBufferLowWatermark() {}
