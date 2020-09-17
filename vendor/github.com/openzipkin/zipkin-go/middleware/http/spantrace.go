// Copyright 2019 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"crypto/tls"
	"net/http/httptrace"
	"strconv"
	"strings"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
)

type spanTrace struct {
	zipkin.Span
	c *httptrace.ClientTrace
}

func (s *spanTrace) getConn(hostPort string) {
	s.Annotate(time.Now(), "Connecting")
	s.Tag("httptrace.get_connection.host_port", hostPort)
}

func (s *spanTrace) gotConn(info httptrace.GotConnInfo) {
	s.Annotate(time.Now(), "Connected")
	s.Tag("httptrace.got_connection.reused", strconv.FormatBool(info.Reused))
	s.Tag("httptrace.got_connection.was_idle", strconv.FormatBool(info.WasIdle))
	if info.WasIdle {
		s.Tag("httptrace.got_connection.idle_time", info.IdleTime.String())
	}
}

func (s *spanTrace) putIdleConn(err error) {
	s.Annotate(time.Now(), "Put Idle Connection")
	if err != nil {
		s.Tag("httptrace.put_idle_connection.error", err.Error())
	}
}

func (s *spanTrace) gotFirstResponseByte() {
	s.Annotate(time.Now(), "First Response Byte")
}

func (s *spanTrace) got100Continue() {
	s.Annotate(time.Now(), "Got 100 Continue")
}

func (s *spanTrace) dnsStart(info httptrace.DNSStartInfo) {
	s.Annotate(time.Now(), "DNS Start")
	s.Tag("httptrace.dns_start.host", info.Host)
}

func (s *spanTrace) dnsDone(info httptrace.DNSDoneInfo) {
	s.Annotate(time.Now(), "DNS Done")
	var addrs []string
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.String())
	}
	s.Tag("httptrace.dns_done.addrs", strings.Join(addrs, " , "))
	if info.Err != nil {
		s.Tag("httptrace.dns_done.error", info.Err.Error())
	}
}

func (s *spanTrace) connectStart(network, addr string) {
	s.Annotate(time.Now(), "Connect Start")
	s.Tag("httptrace.connect_start.network", network)
	s.Tag("httptrace.connect_start.addr", addr)
}

func (s *spanTrace) connectDone(network, addr string, err error) {
	s.Annotate(time.Now(), "Connect Done")
	s.Tag("httptrace.connect_done.network", network)
	s.Tag("httptrace.connect_done.addr", addr)
	if err != nil {
		s.Tag("httptrace.connect_done.error", err.Error())
	}
}

func (s *spanTrace) tlsHandshakeStart() {
	s.Annotate(time.Now(), "TLS Handshake Start")
}

func (s *spanTrace) tlsHandshakeDone(_ tls.ConnectionState, err error) {
	s.Annotate(time.Now(), "TLS Handshake Done")
	if err != nil {
		s.Tag("httptrace.tls_handshake_done.error", err.Error())
	}
}

func (s *spanTrace) wroteHeaders() {
	s.Annotate(time.Now(), "Wrote Headers")
}

func (s *spanTrace) wait100Continue() {
	s.Annotate(time.Now(), "Wait 100 Continue")
}

func (s *spanTrace) wroteRequest(info httptrace.WroteRequestInfo) {
	s.Annotate(time.Now(), "Wrote Request")
	if info.Err != nil {
		s.Tag("httptrace.wrote_request.error", info.Err.Error())
	}
}
