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

package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"mosn.io/mosn/pkg/admin/store"
)

func TestKnownFeatures(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/v1/features", nil)
	w := httptest.NewRecorder()
	knownFeatures(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("response status got %d", resp.StatusCode)
	}
	b, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("response read error: %v", err)
	}
	m := map[string]bool{}
	json.Unmarshal(b, &m)
	v, ok := m[string(store.ConfigAutoWrite)]
	if !ok || v {
		t.Fatalf("features is not expected")
	}
}

func TestSingleFeatureState(t *testing.T) {
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/v1/features?key=auto_config", nil)
	w := httptest.NewRecorder()
	knownFeatures(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("response status got %d", resp.StatusCode)
	}
	b, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("response read error: %v", err)
	}
	if string(b) != "false" {
		t.Fatalf("feature state is not expected: %s", string(b))
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("t1", "test")
	os.Setenv("t3", "")
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/v1/env?key=t1&key=t2&key=t3", nil)
	w := httptest.NewRecorder()
	getEnv(w, r)
	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("response status got %d", resp.StatusCode)
	}
	b, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("response read error: %v", err)
	}
	out := &envResults{}
	expected := &envResults{
		Env: map[string]string{
			"t1": "test",
			"t3": "",
		},
		NotFound: []string{"t2"},
	}
	json.Unmarshal(b, out)
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("env got %s", string(b))
	}
}

// Common Invalid Case
func TestInvalidCommon(t *testing.T) {
	teasCases := []struct {
		Method             string
		Url                string
		ExpectedStatusCode int
		Func               func(w http.ResponseWriter, r *http.Request)
	}{
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/config_dump",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               configDump,
		},
		{
			Method:             "GET",
			Url:                "http://127.0.0.1/api/v1/config_dump?mosnconfig&router",
			ExpectedStatusCode: 400,
			Func:               configDump,
		},
		{
			Method:             "GET",
			Url:                "http://127.0.0.1/api/v1/config_dump?invalid",
			ExpectedStatusCode: 500,
			Func:               configDump,
		},
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/stats",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               statsDump,
		},
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/stats_glob",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               statsDumpProxyTotal,
		},
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/get_loglevel",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               getLoggerInfo,
		},
		{
			Method:             "GET",
			Url:                "http://127.0.0.1/api/v1/update_loglevel",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               updateLogLevel,
		},
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/update_loglevel",
			ExpectedStatusCode: http.StatusBadRequest,
			Func:               updateLogLevel,
		},
		{
			Method:             "GET",
			Url:                "http://127.0.0.1/api/v1/enable_log",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               enableLogger,
		},
		{
			Method:             "GET",
			Url:                "http://127.0.0.1/api/v1/disable_log",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               disableLogger,
		},
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/state",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               getState,
		},
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/features",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               knownFeatures,
		},
		{
			Method:             "POST",
			Url:                "http://127.0.0.1/api/v1/env",
			ExpectedStatusCode: http.StatusMethodNotAllowed,
			Func:               getEnv,
		},
		{
			Method:             "GET",
			Url:                "http://127.0.0.1/api/v1/env",
			ExpectedStatusCode: http.StatusBadRequest,
			Func:               getEnv,
		},
	}
	for idx, tc := range teasCases {
		r := httptest.NewRequest(tc.Method, tc.Url, nil)
		w := httptest.NewRecorder()
		tc.Func(w, r)
		if w.Result().StatusCode != tc.ExpectedStatusCode {
			t.Fatalf("case %d response status code is %d, wanna: %d", idx, w.Result().StatusCode, tc.ExpectedStatusCode)
		}
	}
}
