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
	"io/ioutil"
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
		ServiceName: "http_client",
		IPv4:        net.ParseIP("127.0.0.1"),
		IPv6:        nil,
		Port:        1,
	}
	tracer, err := zipkin.NewTracer(httpReport, zipkin.WithSampler(zipkin.AlwaysSample), zipkin.WithLocalEndpoint(localEndpint))
	if err != nil {
		log.Fatalf("create tracer error: %v", err)
	}
	span := tracer.StartSpan("client request", zipkin.Kind(model.Client), zipkin.StartTime(time.Now()))
	span.Tag(trace.CallerAppName.String(), "ClientApp")

	// execute http request
	req, _ := http.NewRequest("GET", "http://localhost:2046/hello", nil)
	err = b3.InjectHTTP(req)(span.Context())
	if err != nil {
		log.Fatalf("injet http error: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("do request error: %v", resp)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	log.Printf("response: %v", string(data))

	span.Finish()

	// util trigger http report
	time.Sleep(2 * time.Second)
}
