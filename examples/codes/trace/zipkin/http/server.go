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
	"net/http"

	zipkintracer "github.com/openzipkin/zipkin-go"
	middleware "github.com/openzipkin/zipkin-go/middleware/http"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
)

func serveHTTP(w http.ResponseWriter, r *http.Request) {

}

type emptyHandler struct {
}

func (h emptyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}

func main() {
	endpoint, err := zipkintracer.NewEndpoint("server", "127.0.0.1:8081")
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

	http.HandleFunc("/", serveHTTP)
	http.ListenAndServe("127.0.0.1:8081", middleware.NewServerMiddleware(tracer)(emptyHandler{}))
}
