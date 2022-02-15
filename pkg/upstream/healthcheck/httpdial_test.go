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

package healthcheck

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"mosn.io/api"
)

func Test_NewSession(t *testing.T) {
	hdsf := &HTTPDialSessionFactory{}
	h1 := &mockHost{}
	h1.addr = "127.0.0.1"
	cfg := make(map[string]interface{})
	hcs := hdsf.NewSession(cfg, h1)
	if reflect.TypeOf(hcs) != reflect.TypeOf(&TCPDialSession{}) {
		t.Errorf("Test_NewSession %+v", hcs)
	}

	h1.addr = "127.0.0.1:22222"
	hcs = hdsf.NewSession(cfg, h1)
	if reflect.TypeOf(hcs) != reflect.TypeOf(&TCPDialSession{}) {
		t.Errorf("Test_NewSession %+v", hcs)
	}

	cfg[HTTPCheckConfigKey] = "xx"
	hcs = hdsf.NewSession(cfg, h1)
	if hcs != nil {
		t.Errorf("Test_NewSession %+v", hcs)
	}

	cfg[HTTPCheckConfigKey] = &HttpCheckConfig{
		Port: -1,
	}
	hcs = hdsf.NewSession(cfg, h1)
	hds := hcs.(*HTTPDialSession)
	if hds.checkUrl != "http://127.0.0.1:22222" {
		t.Errorf("Test_NewSession %+v", hcs)
	}

	cfg[HTTPCheckConfigKey] = &HttpCheckConfig{
		Port: 33333,
	}
	hcs = hdsf.NewSession(cfg, h1)
	hds = hcs.(*HTTPDialSession)
	if hcs == nil {
		t.Errorf("Test_NewSession %+v", hcs)
	}

	if hds.checkUrl != "http://127.0.0.1:33333" {
		t.Errorf("Test_NewSession %+v", hds)
	}
	if hds.timeout != time.Second*30 {
		t.Errorf("Test_NewSession %+v", hds)
	}

	cfg[HTTPCheckConfigKey] = &HttpCheckConfig{
		Port:    33333,
		Timeout: api.DurationConfig{time.Second * 15},
		Path:    "/test",
	}

	hcs = hdsf.NewSession(cfg, h1)
	hds = hcs.(*HTTPDialSession)
	if hcs == nil {
		t.Errorf("Test_NewSession %+v", hcs)
	}

	if hds.checkUrl != "http://127.0.0.1:33333/test" {
		t.Errorf("Test_NewSession %+v", hds)
	}
	if hds.timeout != time.Second*15 {
		t.Errorf("Test_NewSession %+v", hds)
	}

	h1.addr = "127.0.0.1"
	hcs = hdsf.NewSession(cfg, h1)
	if hcs != nil {
		t.Errorf("Test_NewSession %+v", hcs)
	}
}

func Test_CheckHealth(t *testing.T) {
	hdsf := &HTTPDialSessionFactory{}
	h1 := &mockHost{}
	h1.addr = "127.0.0.1:22222"
	cfg := make(map[string]interface{})
	cfg[HTTPCheckConfigKey] = &HttpCheckConfig{
		Port: 33333,
	}
	hcs := hdsf.NewSession(cfg, h1)
	hds := hcs.(*HTTPDialSession)
	h := hds.CheckHealth()
	if h {
		t.Errorf("Test_CheckHealth Error")
	}

	hds.client = &http.Client{}

	code := 200
	server := &http.Server{Addr: "127.0.0.1:33333"}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(code)
	})
	go server.ListenAndServe()
	time.Sleep(time.Second)

	h = hds.CheckHealth()
	if !h {
		t.Errorf("Test_CheckHealth Error")
	}

	code = 500
	h = hds.CheckHealth()
	if h {
		t.Errorf("Test_CheckHealth Error")
	}

	server.Close()
}
