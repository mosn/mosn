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

package zipkin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	"mosn.io/api"
	"mosn.io/mosn/pkg/trace/sofa/xprotocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
)

const (
	outbound                = "outbound"
	spanTagServiceName      = "serviceName"
	spanTagServiceNamespace = "serviceNamespace"
	spanTagServiceVersion   = "serviceVersion"
)

type Span struct {
	carries            Carrier
	startTime, endTime time.Time
	tags               [xprotocol.TRACE_END]string
	tracer             *zipkin.Tracer
	span               zipkin.Span
}

func NewSpan(ctx *model.SpanContext, tracer *zipkin.Tracer, carries Carrier, startTime time.Time) *Span {
	opts := []zipkin.SpanOption{
		zipkin.StartTime(startTime),
	}
	if ctx != nil {
		opts = append(opts, zipkin.Parent(*ctx))
	}
	span := tracer.StartSpan("", opts...)
	s := &Span{
		startTime: startTime,
		carries:   carries,
		tracer:    tracer,
		span:      span,
	}
	s.injectCtx()
	return s
}

func (s *Span) TraceId() string {
	return s.Tag(xprotocol.TRACE_ID)
}

func (s *Span) SpanId() string {
	return s.Tag(xprotocol.SPAN_ID)
}

func (s *Span) ParentSpanId() string {
	return s.Tag(xprotocol.PARENT_SPAN_ID)
}

func (s *Span) SetOperation(operation string) {
	s.span.SetName(operation)
}

func (s *Span) SetTag(key uint64, value string) {
	switch key {
	case xprotocol.SERVICE_NAME:
		s.span.Tag("serviceName", value)
	case xprotocol.METHOD_NAME:
		s.span.Tag("methodName", value)
	case xprotocol.PROTOCOL:
		s.span.Tag("protocol", value)
	case xprotocol.RESULT_STATUS:
		s.span.Tag("resultStatus", value)
	case xprotocol.REQUEST_SIZE:
		s.span.Tag("requestSize", value)
	case xprotocol.RESPONSE_SIZE:
		s.span.Tag("responseSize", value)
	case xprotocol.UPSTREAM_HOST_ADDRESS:
		s.span.Tag("upstreamHostAddress", value)
	case xprotocol.DOWNSTEAM_HOST_ADDRESS:
		s.span.Tag("downstreamHostAddress", value)
	case xprotocol.APP_NAME:
		s.span.Tag("appName", value)
	case xprotocol.TARGET_APP_NAME:
		s.span.Tag("targetAppName", value)
	case xprotocol.SPAN_TYPE:
		s.span.Tag("spanType", value)
	case xprotocol.BAGGAGE_DATA:
		s.span.Tag("baggageData", value)
	case xprotocol.REQUEST_URL:
		s.span.Tag("requestUrl", value)
	case xprotocol.TARGET_CELL:
		s.span.Tag("targetCell", value)
	case xprotocol.TARGET_IDC:
		s.span.Tag("targetIdc", value)
	case xprotocol.TARGET_CITY:
		s.span.Tag("targetCity", value)
	case xprotocol.ROUTE_RECORD:
		s.span.Tag("routeRecord", value)
	}
	s.tags[key] = value
}

func (s *Span) SetRequestInfo(reqinfo api.RequestInfo) {
	if reqinfo == nil {
		return
	}
	routeEntrt := reqinfo.RouteEntry()
	if routeEntrt != nil {
		clusterName := routeEntrt.ClusterName()
		if clusterName != "" {
			info := paresClusterName(clusterName)
			if info.Version != "" {
				s.span.Tag(spanTagServiceVersion, info.Version)
			}
			if info.Name != "" {
				s.span.Tag(spanTagServiceName, info.Name)
			}
			if info.Namespace != "" {
				s.span.Tag(spanTagServiceNamespace, info.Namespace)
			}
		}
	}

	requestSize := strconv.FormatUint(reqinfo.BytesReceived(), 10)
	s.SetTag(xprotocol.REQUEST_SIZE, requestSize)
	responseSize := strconv.FormatUint(reqinfo.BytesSent(), 10)
	s.SetTag(xprotocol.RESPONSE_SIZE, responseSize)

	if reqinfo.UpstreamHost() != nil {
		s.SetTag(xprotocol.UPSTREAM_HOST_ADDRESS, reqinfo.UpstreamHost().AddressString())
	}
	if reqinfo.DownstreamLocalAddress() != nil {
		s.SetTag(xprotocol.DOWNSTEAM_HOST_ADDRESS, reqinfo.DownstreamRemoteAddress().String())
	}
	s.SetTag(xprotocol.RESULT_STATUS, strconv.Itoa(reqinfo.ResponseCode()))
	s.SetTag(xprotocol.PROTOCOL, string(reqinfo.Protocol()))
	s.span.SetName(fmt.Sprintf("%s", string(reqinfo.Protocol())))
}

func (s *Span) Tag(key uint64) string {
	return s.tags[key]
}

func (s *Span) FinishSpan() {
	s.endTime = time.Now()
	s.span.Finish()
}

func (s *Span) InjectContext(requestHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	carries := &HeaderCarrier{requestHeaders}
	s.injectContext(carries)
}

func (s *Span) injectContext(set Setter) {
	err := InjectB3(set, s.span.Context())
	if err != nil {
		log.DefaultLogger.Warnf("zipkin: InjectContext: %s", err.Error())
	}
}

func (s *Span) injectCtx() {
	if s.carries == nil {
		return
	}
	s.injectContext(s.carries)
	s.carries.ForeachKey(func(key, val string) error {
		s.span.Tag(key, val)
		return nil
	})

	if spanId := s.carries.Get(b3.SpanID); spanId != "" {
		s.SetTag(xprotocol.SPAN_ID, spanId)
	}
	if parentSpanId := s.carries.Get(b3.ParentSpanID); parentSpanId != "" {
		s.SetTag(xprotocol.PARENT_SPAN_ID, parentSpanId)
	}
	if traceId := s.carries.Get(b3.TraceID); traceId != "" {
		s.SetTag(xprotocol.TRACE_ID, traceId)
	}
}

func (s *Span) SpawnChild(operationName string, startTime time.Time) types.Span {
	return nil
}

func (s *Span) SetStartTime(startTime time.Time) {
	s.startTime = startTime
}

func (s *Span) String() string {
	return fmt.Sprintf("TraceId:%s;SpanId:%s;Duration:%s;Protocol:%s;ServiceName:%s;requestSize:%s;responseSize:%s;upstreamHostAddress:%s;downstreamRemoteHostAdress:%s",
		s.tags[xprotocol.TRACE_ID],
		s.tags[xprotocol.SPAN_ID],
		strconv.FormatInt(s.endTime.Sub(s.startTime).Nanoseconds()/1000000, 10),
		s.tags[xprotocol.PROTOCOL],
		s.tags[xprotocol.SERVICE_NAME],
		s.tags[xprotocol.REQUEST_SIZE],
		s.tags[xprotocol.RESPONSE_SIZE],
		s.tags[xprotocol.UPSTREAM_HOST_ADDRESS],
		s.tags[xprotocol.DOWNSTEAM_HOST_ADDRESS])
}

type clusterNameInfo struct {
	Version   string
	Name      string
	Namespace string
}

func paresClusterName(clusterName string) *clusterNameInfo {
	c := &clusterNameInfo{}
	items := strings.Split(clusterName, "|")
	if len(items) == 4 {
		if outbound == items[0] {
			c.Version = items[2]
		}

		service := items[3]
		serviceItems := strings.SplitN(service, ".", 3)
		if len(serviceItems) == 3 {
			c.Name = serviceItems[0]
			c.Namespace = serviceItems[1]
		}
	}
	return c
}
