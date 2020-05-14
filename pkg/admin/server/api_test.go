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
