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

package http

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	zipkintracer "github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"mosn.io/api"
	"mosn.io/mosn/pkg/trace/zipkin"
)

func NewTracer(zipkinServer, serviceName, instanceIP string) (*Tracer, error) {
	endpoint, err := zipkintracer.NewEndpoint(serviceName, instanceIP)
	if err != nil {
		return nil, err
	}

	reporter := zipkinhttp.NewReporter(zipkinServer,
		zipkinhttp.BatchSize(0),
	)
	tracer, err := zipkintracer.NewTracer(reporter,
		zipkintracer.WithLocalEndpoint(endpoint),
		zipkintracer.WithSharedSpans(true),
		zipkintracer.WithTraceID128Bit(true),
	)

	if err != nil {
		return nil, err
	}
	return &Tracer{
		tracer: tracer,
	}, nil
}

func TestTracer(t *testing.T) {
	var mockNode5 = makeMockServer("node_5", nil)
	var mockNode4 = makeMockServer("node_4", makeCaller(mockNode5.URL))
	var mockNode3 = makeMockServer("node_3", makeCaller(mockNode5.URL))
	var mockNode2 = makeMockServer("node_2", makeCaller(mockNode4.URL, mockNode5.URL, mockNode3.URL, mockNode5.URL))
	var mockNode1 = makeMockServer("node_1", makeCaller(mockNode2.URL, mockNode5.URL))
	var mockNode0 = makeMockServer("node_0", makeCaller(mockNode1.URL))
	servers := []*httptest.Server{
		mockNode0,
		mockNode1,
		mockNode2,
		mockNode4,
		mockNode3,
		mockNode5,
	}
	req, err := http.NewRequest(http.MethodGet, mockNode0.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	resp.Body.Close()

	for _, s := range servers {
		s.Close()
	}
	mockZipkin.Close()
}

var wait = make(chan struct{}, 1)
var mockZipkin = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	fmt.Println(string(body))

	rw.Write(body)
}))

// var zipkinURL = "http://127.0.0.1:9411/api/v2/spans"
var zipkinURL = mockZipkin.URL + "/api/v2/spans"

func makeCaller(targers ...string) func(span *zipkin.Span) error {
	return func(span *zipkin.Span) error {
		for _, targer := range targers {
			req, err := http.NewRequest(http.MethodGet, targer, nil)
			if err != nil {
				return err
			}

			span.InjectContext(httpHeaderCarrier(req.Header), nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			resp.Body.Close()
		}
		return nil
	}
}

func makeMockServer(serviceName string, cb func(span *zipkin.Span) error) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		tracer, err := NewTracer(zipkinURL, serviceName, "127.0.0.1")
		if err != nil {
			log.Println(err)
		}

		span := tracer.Start(r.Context(), httpHeaderCarrier(r.Header), time.Now())
		span.SetOperation("/" + r.Method + r.URL.String())
		defer span.FinishSpan()
		if cb != nil {
			err := cb(span.(*zipkin.Span))
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
			}
		}
	}))
}

type httpHeaderCarrier http.Header

func (c httpHeaderCarrier) Get(key string) (string, bool) {
	h := http.Header(c)
	if _, ok := h[key]; !ok {
		return "", false
	}
	return h.Get(key), true
}

func (c httpHeaderCarrier) Set(key, val string) {
	h := http.Header(c)
	h.Set(key, val)
}

func (c httpHeaderCarrier) Add(key, value string) {
	h := http.Header(c)
	h.Add(key, value)
}

func (c httpHeaderCarrier) Del(key string) {
	h := http.Header(c)
	h.Del(key)
}

func (c httpHeaderCarrier) Range(f func(key, value string) bool) {
	for k, vals := range c {
		if len(vals) > 0 {
			if c := f(k, vals[0]); !c {
				return
			}
		}
	}
}

func (c httpHeaderCarrier) Clone() api.HeaderMap {
	return c
}

func (c httpHeaderCarrier) ByteSize() uint64 {
	return 0
}
