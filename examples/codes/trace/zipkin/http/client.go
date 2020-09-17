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
	"fmt"
	"log"
	"net/http"
	"time"

	zipkintracer "github.com/openzipkin/zipkin-go"
	middleware "github.com/openzipkin/zipkin-go/middleware/http"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
)

func main() {
	endpoint, err := zipkintracer.NewEndpoint("client", "127.0.0.1")
	if err != nil {
		log.Fatalf("create tracer error %v \n", err)
	}

	reporter := zipkinhttp.NewReporter("http://127.0.0.1:9411/api/v2/spans")

	tracer, err := zipkintracer.NewTracer(reporter,
		zipkintracer.WithLocalEndpoint(endpoint),
		zipkintracer.WithSharedSpans(false),
		zipkintracer.WithTraceID128Bit(true),
	)
	if err != nil {
		log.Fatalf("create tracer error %v \n", err)
	}

	client, err := middleware.NewClient(tracer)
	if err != nil {
		log.Fatalf("create tracer error %v \n", err)
	}

	// call server
	request, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:2046/zipkin"), nil)
	if err != nil {
		log.Fatalf("unable to create http request: %+v\n", err)
	}
	for {
		res, err := client.Do(request)
		if err != nil {
			log.Fatalf("unable to do http request: %+v\n", err)
		}
		_ = res.Body.Close()
		time.Sleep(time.Second)
	}
}
