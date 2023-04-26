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
	if hds.request.URL.String() != "http://127.0.0.1:22222" {
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

	if hds.request.URL.String() != "http://127.0.0.1:33333" {
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

	if hds.request.URL.String() != "http://127.0.0.1:33333/test" {
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

	h1.addr = "127.0.0.1:44444"
	cfg[HTTPCheckConfigKey] = &HttpCheckConfig{
		Port:    44444,
		Timeout: api.DurationConfig{time.Second * 15},
		Path:    "/test",
		Domain:  "test.healthcheck.com",
		Method:  "HEAD",
		Scheme:  "https",
	}

	hcs = hdsf.NewSession(cfg, h1)
	hds = hcs.(*HTTPDialSession)
	if hcs == nil {
		t.Errorf("Test_NewSession %+v", hcs)
	}
	if hds.request.URL.String() != "https://127.0.0.1:44444/test" {
		t.Errorf("Test_NewSession %+v", hds)
	}
	if hds.request.URL.Scheme != "https" {
		t.Errorf("Test_NewSession scheme should be https, %+v", hds)
	}
	if hds.request.Method != "HEAD" {
		t.Errorf("Test_NewSession method should be HEAD, %+v", hds)
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

func Test_CheckL7Health(t *testing.T) {
	httpAddr := "127.0.0.1:22222"
	testCases := []struct {
		name       string
		serverCode int
		serverPath string
		hostAddr   string
		config     HttpCheckConfig
		expect     bool
	}{
		{
			name:       "connect failed",
			serverPath: "/index.html",
			hostAddr:   "127.0.0.1:22224",
			config: HttpCheckConfig{
				Path: "/index.html",
			},
			expect: false,
		},
		{
			name:       "normal_code",
			serverCode: 200,
			serverPath: "/200",
			hostAddr:   httpAddr,
			config: HttpCheckConfig{
				Path: "/200",
			},
			expect: true,
		},
		{
			name:     "normal_code_with_querystring",
			hostAddr: httpAddr,
			config: HttpCheckConfig{
				Path: "/200?test=foo&test1=bar",
			},
			expect: true,
		},
		{
			name:       "normal_code_only_querystring",
			serverCode: 200,
			serverPath: "/",
			hostAddr:   httpAddr,
			config: HttpCheckConfig{
				Path: "?test=foo&test1=bar",
			},
			expect: true,
		},
		{
			name:     "normal_code_no_path_and_querystring",
			hostAddr: httpAddr,
			config: HttpCheckConfig{
				Path: "?",
			},
			expect: true,
		},
		{
			name:     "normal_code_force_query",
			hostAddr: httpAddr,
			config: HttpCheckConfig{
				Path: "/200?",
			},
			expect: true,
		},
		{
			name:     "abnormal_code_illegal_path",
			hostAddr: httpAddr,
			config: HttpCheckConfig{
				Path: "http://localhost/400?foo=bar",
				Codes: []CodeRange{
					{
						Start: 300,
						End:   401,
					},
				},
			},
			expect: false,
		},
		{
			name:       "abnormal_code",
			serverCode: 500,
			serverPath: "/500",
			hostAddr:   httpAddr,
			config: HttpCheckConfig{
				Path: "/500",
			},
			expect: false,
		},
		{
			name:       "match_code_range",
			serverCode: 400,
			serverPath: "/400",
			hostAddr:   httpAddr,
			config: HttpCheckConfig{
				Path: "/400",
				Codes: []CodeRange{
					{
						Start: 200,
						End:   500,
					},
				},
			},
			expect: true,
		},
		{
			name:       "https_scheme_fail",
			serverCode: 200,
			serverPath: "/https_fail",
			hostAddr:   httpAddr,
			config: HttpCheckConfig{
				Path:   "/https_tail",
				Scheme: "https",
			},
			expect: false,
		},
	}

	mux := http.NewServeMux()
	addPath := func(path string, code int) {
		mux.HandleFunc(path, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(code)
		})
	}

	// start httpserver
	go http.ListenAndServe("127.0.0.1:22222", mux)
	time.Sleep(time.Second)

	for _, tc := range testCases {
		hdsf := &HTTPDialSessionFactory{}
		h1 := &mockHost{}
		h1.addr = tc.hostAddr
		cfg := make(map[string]interface{})
		cfg[HTTPCheckConfigKey] = &tc.config
		hcs := hdsf.NewSession(cfg, h1)
		hds := hcs.(*HTTPDialSession)

		if tc.serverPath != "" {
			addPath(tc.serverPath, tc.serverCode)
		}

		h := hds.CheckHealth()
		if h != tc.expect {
			t.Errorf("Test_CheckHealth Error, case:%s", tc.name)
		}
	}
}
