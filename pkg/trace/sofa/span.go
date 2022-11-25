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

package sofa

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/track"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
)

type SofaRPCSpan struct {
	ctx           context.Context
	ingressLogger *log.Logger
	egressLogger  *log.Logger
	startTime     time.Time
	endTime       time.Time
	tags          [TRACE_END]string
	traceId       string
	spanId        string
	parentSpanId  string
	operationName string
}

func (s *SofaRPCSpan) TraceId() string {
	return s.traceId
}

func (s *SofaRPCSpan) SpanId() string {
	return s.spanId
}

func (s *SofaRPCSpan) ParentSpanId() string {
	return s.parentSpanId
}

func (s *SofaRPCSpan) SetOperation(operation string) {
	s.operationName = operation
}

func (s *SofaRPCSpan) SetTag(key uint64, value string) {
	if key == TRACE_ID {
		s.traceId = value
	} else if key == SPAN_ID {
		s.spanId = value
	} else if key == PARENT_SPAN_ID {
		s.parentSpanId = value
	}

	s.tags[key] = value
}

func (s *SofaRPCSpan) SetRequestInfo(reqinfo api.RequestInfo) {
	s.tags[REQUEST_SIZE] = strconv.FormatInt(int64(reqinfo.BytesReceived()), 10)
	s.tags[RESPONSE_SIZE] = strconv.FormatInt(int64(reqinfo.BytesSent()), 10)
	if reqinfo.UpstreamHost() != nil {
		s.tags[UPSTREAM_HOST_ADDRESS] = reqinfo.UpstreamHost().AddressString()
	}
	if reqinfo.DownstreamRemoteAddress() != nil {
		s.tags[DOWNSTEAM_HOST_ADDRESS] = reqinfo.DownstreamRemoteAddress().String()
	}
	s.tags[RESULT_STATUS] = strconv.Itoa(reqinfo.ResponseCode())
	s.tags[MOSN_PROCESS_TIME] = reqinfo.ProcessTimeDuration().String()
	s.tags[MOSN_PROCESS_FAIL] = strconv.FormatBool(reqinfo.GetResponseFlag(types.MosnProcessFailedFlags))
}

func (s *SofaRPCSpan) Tag(key uint64) string {
	return s.tags[key]
}

func (s *SofaRPCSpan) FinishSpan() {
	s.endTime = time.Now()
	err := s.log()
	if err == types.ErrChanFull {
		if log.DefaultLogger.GetLogLevel() >= log.WARN {
			log.DefaultLogger.Warnf("Channel is full, discard span, trace id is " + s.traceId + ", span id is " + s.spanId)
		}
	}
}

func (s *SofaRPCSpan) InjectContext(requestHeaders api.HeaderMap, requestInfo api.RequestInfo) {
}

func (s *SofaRPCSpan) SpawnChild(operationName string, startTime time.Time) api.Span {
	return nil
}

func (s *SofaRPCSpan) SetStartTime(startTime time.Time) {
	s.startTime = startTime
}

func (s *SofaRPCSpan) String() string {
	return fmt.Sprintf("TraceId:%s;SpanId:%s;Duration:%s;ProtocolName:%s;ServiceName:%s;requestSize:%s;responseSize:%s;upstreamHostAddress:%s;downstreamRemoteHostAdress:%s",
		s.tags[TRACE_ID],
		s.tags[SPAN_ID],
		strconv.FormatInt(s.endTime.Sub(s.startTime).Nanoseconds()/1000000, 10),
		s.tags[PROTOCOL],
		s.tags[SERVICE_NAME],
		s.tags[REQUEST_SIZE],
		s.tags[RESPONSE_SIZE],
		s.tags[UPSTREAM_HOST_ADDRESS],
		s.tags[DOWNSTEAM_HOST_ADDRESS])
}

func (s *SofaRPCSpan) EndTime() time.Time {
	return s.endTime
}

func (s *SofaRPCSpan) StartTime() time.Time {
	return s.startTime
}

func (s *SofaRPCSpan) log() error {
	printData := log.GetLogBuffer(512)
	printData.WriteString("{")
	printData.WriteString("\"timestamp\":")
	date := s.endTime.Format("2006-01-02 15:04:05.000")
	printData.WriteString("\"" + date + "\",")

	printData.WriteString("\"traceId\":")
	printData.WriteString("\"" + s.tags[TRACE_ID] + "\",")

	printData.WriteString("\"spanId\":")
	printData.WriteString("\"" + s.tags[SPAN_ID] + "\",")

	printData.WriteString("\"service\":")
	printData.WriteString("\"" + s.tags[SERVICE_NAME] + "\",")

	printData.WriteString("\"method\":")
	printData.WriteString("\"" + s.tags[METHOD_NAME] + "\",")

	printData.WriteString("\"protocol\":")
	printData.WriteString("\"" + s.tags[PROTOCOL] + "\",")

	printData.WriteString("\"resp.size\":")
	printData.WriteString("\"" + s.tags[RESPONSE_SIZE] + "\",")

	printData.WriteString("\"req.size\":")
	printData.WriteString("\"" + s.tags[REQUEST_SIZE] + "\",")

	printData.WriteString("\"baggage\":")
	printData.WriteString("\"" + s.tags[BAGGAGE_DATA] + "\",")

	printData.WriteString("\"mosn.duration\":")
	printData.WriteString("\"" + s.tags[MOSN_PROCESS_TIME] + "\",")

	tracks := track.TrackBufferByContext(s.ctx).Tracks

	printData.WriteString("\"mosn.duration.detail\":")
	printData.WriteString("\"" + tracks.GetTrackCosts() + "\",")

	printData.WriteString("\"mosn.data.timestamp\":")
	printData.WriteString("\"" + tracks.GetTrackTimestamp() + "\",")

	// Set status code. TODO can not get the result code if server throw an exception.

	statusCode, _ := strconv.Atoi(s.tags[RESULT_STATUS])
	var code = "02"
	if statusCode == api.SuccessCode {
		code = "00"
	} else if statusCode == api.TimeoutExceptionCode {
		code = "03"
	} else if statusCode == api.RouterUnavailableCode || statusCode == api.NoHealthUpstreamCode {
		code = "04"
	} else {
		code = "02"
	}

	printData.WriteString("\"result.code\":")
	printData.WriteString("\"" + code + "\",")

	if s.tags[SPAN_TYPE] == "ingress" {
		kind := "server"

		printData.WriteString("\"span.kind\":")
		printData.WriteString("\"" + kind + "\",")

		printData.WriteString("\"remote.ip\":")
		printData.WriteString("\"" + s.tags[DOWNSTEAM_HOST_ADDRESS] + "\",")

		printData.WriteString("\"remote.app\":")
		printData.WriteString("\"" + s.tags[CALLER_APP_NAME] + "\",")

		printData.WriteString("\"local.app\":")
		printData.WriteString("\"" + "TODO" + "\",") //TODO

		// The time server(upstream) takes to process the RPC
		// server.duration = server.pool.wait.time + biz.impl.time + resp.serialize.time + req.deserialize.time
		duration := strconv.FormatInt(s.endTime.Sub(s.startTime).Nanoseconds()/1000000, 10)

		printData.WriteString("\"server.duration\":")
		printData.WriteString("\"" + duration + "\"")
		printData.WriteString("}")
		printData.WriteString("\n")

		return s.ingressLogger.Print(printData, true)
	}

	if s.tags[SPAN_TYPE] == "egress" {
		kind := "client"

		printData.WriteString("\"span.kind\":")
		printData.WriteString("\"" + kind + "\",")

		invokeType := "sync"
		printData.WriteString("\"invoke.type\":")
		printData.WriteString("\"" + invokeType + "\",") //TODO

		routerRecord := ""
		printData.WriteString("\"router.record\":")
		printData.WriteString("\"" + routerRecord + "\",") //TODO

		printData.WriteString("\"remote.ip\":")
		printData.WriteString("\"" + s.tags[UPSTREAM_HOST_ADDRESS] + "\",")

		downStreamHostAddress := strings.Split(s.tags[DOWNSTEAM_HOST_ADDRESS], ":")
		if len(downStreamHostAddress) > 0 {
			localIp := strings.Split(s.tags[DOWNSTEAM_HOST_ADDRESS], ":")[0]
			printData.WriteString("\"local.client.ip\":")
			printData.WriteString("\"" + localIp + "\",")
		}
		if len(downStreamHostAddress) > 1 {
			localPort := strings.Split(s.tags[DOWNSTEAM_HOST_ADDRESS], ":")[1]
			printData.WriteString("\"local.client.port\":")
			printData.WriteString("\"" + localPort + "\",")
		}
		elapse := strconv.FormatInt(s.endTime.Sub(s.startTime).Nanoseconds()/1000000, 10)
		printData.WriteString("\"client.elapse.time\":")
		printData.WriteString("\"" + elapse + "\"")
		printData.WriteString("}")
		printData.WriteString("\n")
		return s.egressLogger.Print(printData, true)
	}
	return nil
}

func NewSpan(ctx context.Context, startTime time.Time) *SofaRPCSpan {
	return &SofaRPCSpan{
		ctx:       ctx,
		startTime: startTime,
	}
}
