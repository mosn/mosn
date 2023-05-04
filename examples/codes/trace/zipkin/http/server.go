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

package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
	zhttp "github.com/openzipkin/zipkin-go/reporter/http"
	trace "mosn.io/mosn/pkg/trace/zipkin"
)

func main() {
	httpReport := zhttp.NewReporter("http://localhost:9411/api/v2/spans", zhttp.BatchSize(1))
	localEndpint := &model.Endpoint{
		ServiceName: "http_server",
		IPv4:        net.ParseIP("127.0.0.1"),
		IPv6:        nil,
		Port:        8080,
	}
	tracer, err := zipkin.NewTracer(httpReport, zipkin.WithSampler(zipkin.AlwaysSample), zipkin.WithLocalEndpoint(localEndpint))
	if err != nil {
		log.Fatalf("create tracer error: %v", err)
	}
	http.HandleFunc("/hello", func(writer http.ResponseWriter, request *http.Request) {
		sc := tracer.Extract(b3.ExtractHTTP(request))
		span := tracer.StartSpan("server response", zipkin.Kind(model.Client), zipkin.StartTime(time.Now()), zipkin.Parent(sc))
		span.Tag(trace.CalleeAppName.String(), "RemoteApp")
		span.Finish()
		writer.Write([]byte("hello world"))
	})

	http.ListenAndServe(":8080", nil)
}
