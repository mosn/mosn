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

package trace

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/json-iterator/go"
)

// -------- SofaTracerSpan --------

type SofaTracerSpan struct {
	tracer        *SofaTracer
	startTime     time.Time
	endTime       time.Time
	tags          map[string]string
	traceId       string
	spanId        string
	parentSpanId  string
	operationName string
}

func (s *SofaTracerSpan) TraceId() string {
	return s.traceId
}

func (s *SofaTracerSpan) SpanId() string {
	return s.spanId
}

func (s *SofaTracerSpan) ParentSpanId() string {
	return s.parentSpanId
}

func (s *SofaTracerSpan) SetOperation(operation string) {
	s.operationName = operation
}

func (s *SofaTracerSpan) SetTag(key string, value string) {
	if key == TRACE_ID {
		s.traceId = value
	} else if key == SPAN_ID {
		s.spanId = value
	} else if key == PARENT_SPAN_ID {
		s.parentSpanId = value
	}

	s.tags[key] = value
}

func (s *SofaTracerSpan) FinishSpan() {
	s.endTime = time.Now()
	select {
	case s.tracer.spanChan <- s:
	default:
		log.DefaultLogger.Warnf("Channel is full, discard span, trace id is " + s.traceId + ", span id is " + s.spanId)
	}
}

func (s *SofaTracerSpan) InjectContext(requestHeaders map[string]string) {
}

func (s *SofaTracerSpan) SpawnChild(operationName string, startTime time.Time) types.Span {
	return nil
}

func (s *SofaTracerSpan) String() string {
	return fmt.Sprintf("TraceId:%s;SpanId:%s;Duration:%s;Protocol:%s;ServiceName:%s;requestSize:%s;responseSize:%s;upstreamHostAddress:%s;downstreamRemoteHostAdress:%s",
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

func (s *SofaTracerSpan) EndTime() time.Time {
	return s.endTime
}

// -------- SofaTracer --------
var SofaTracerInstance *SofaTracer
var PrintLog = true

func CreateInstance() {
	SofaTracerInstance = newSofaTracer()
}

type SofaTracer struct {
	spanChan      chan *SofaTracerSpan
	ingressLogger log.Logger
	egressLogger  log.Logger
}

func newSofaTracer() *SofaTracer {
	instance := &SofaTracer{}
	instance.spanChan = make(chan *SofaTracerSpan, 1000)

	if PrintLog {
		userHome := os.Getenv("HOME")
		var err error
		logRoot := userHome + "/logs/tracelog/mosn/"
		instance.ingressLogger, err = log.NewLogger(logRoot+"rpc-server-digest.log", log.INFO)
		if err != nil {
			// TODO when error is not nil
		}

		instance.egressLogger, err = log.NewLogger(logRoot+"rpc-client-digest.log", log.INFO)
		if err != nil {
			// TODO when error is not nil
		}

		// Do not print any timestamp prefix
		instance.ingressLogger.SetFlags(0)
		instance.egressLogger.SetFlags(0)

		go func() {
			for {
				span := <-instance.spanChan
				instance.printSpan(span)
			}
		}()
	}

	return instance
}

func (tracer *SofaTracer) Start(startTime time.Time) types.Span {
	span := &SofaTracerSpan{
		tracer:    tracer,
		startTime: startTime,
		tags:      map[string]string{},
	}

	return span
}

func (tracer *SofaTracer) GetSpan() types.Span {
	select {
	case span := <-tracer.spanChan:
		return span
	default:
		return nil
	}
}

func (tracer *SofaTracer) printSpan(span *SofaTracerSpan) {
	printData := make(map[string]string)
	printData["timestamp"] = span.endTime.Format("2006-01-02 15:04:05.999")
	printData["traceId"] = span.tags[TRACE_ID]
	printData["spanId"] = span.tags[SPAN_ID]
	printData["service"] = span.tags[SERVICE_NAME]
	printData["method"] = span.tags[METHOD_NAME]
	printData["protocol"] = span.tags[PROTOCOL]
	printData["resp.size"] = span.tags[RESPONSE_SIZE]
	printData["req.size"] = span.tags[REQUEST_SIZE]
	printData["baggage"] = span.tags[BAGGAGE_DATA]

	// Set status code. TODO can not get the result code if server throw an exception.
	statusCode, _ := strconv.Atoi(span.tags[RESULT_STATUS])
	if statusCode == types.SuccessCode {
		printData["result.code"] = "00"
	} else if statusCode == types.TimeoutExceptionCode {
		printData["result.code"] = "03"
	} else if statusCode == types.RouterUnavailableCode || statusCode == types.NoHealthUpstreamCode {
		printData["result.code"] = "04"
	} else {
		printData["result.code"] = "02"
	}

	if span.tags[SPAN_TYPE] == "ingress" {
		printData["span.kind"] = "server"
		printData["remote.ip"] = span.tags[DOWNSTEAM_HOST_ADDRESS]
		printData["remote.app"] = span.tags[APP_NAME]
		printData["local.app"] = "TODO" // TODO
		// The time server(upstream) takes to process the RPC
		// server.duration = server.pool.wait.time + biz.impl.time + resp.serialize.time + req.deserialize.time
		printData["server.duration"] = strconv.FormatInt(span.endTime.Sub(span.startTime).Nanoseconds()/1000000, 10)
		result, _ := jsoniter.MarshalToString(printData)
		tracer.ingressLogger.Println(result)
	}

	if span.tags[SPAN_TYPE] == "egress" {
		printData["span.kind"] = "client"
		printData["invoke.type"] = "sync" // TODO
		printData["router.record"] = ""   // TODO
		printData["remote.ip"] = span.tags[UPSTREAM_HOST_ADDRESS]
		downStreamHostAddress := strings.Split(span.tags[DOWNSTEAM_HOST_ADDRESS], ":")
		if len(downStreamHostAddress) > 0 {
			printData["local.client.ip"] = strings.Split(span.tags[DOWNSTEAM_HOST_ADDRESS], ":")[0]
		}
		if len(downStreamHostAddress) > 1 {
			printData["local.client.port"] = strings.Split(span.tags[DOWNSTEAM_HOST_ADDRESS], ":")[1]
		}
		printData["client.elapse.time"] = strconv.FormatInt(span.endTime.Sub(span.startTime).Nanoseconds()/1000000, 10)
		result, _ := jsoniter.MarshalToString(printData)
		tracer.egressLogger.Println(result)
	}
}
