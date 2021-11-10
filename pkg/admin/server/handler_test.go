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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultAPIHandler(t *testing.T) {
	success := 0
	handler := func(w http.ResponseWriter, r *http.Request) {
		success++
	}
	t.Run("default without any auths", func(t *testing.T) {
		success = 0
		apiHandler := NewAPIHandler(handler)
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "http://127.0.0.1/handler/test", nil)
		apiHandler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 1, success)
	})
	t.Run("default failed, with auth", func(t *testing.T) {
		success = 0
		apiHandler := NewAPIHandler(handler, NewAuth(func(r *http.Request) bool {
			return r.FormValue("pass") == "true"
		}, nil))
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "http://127.0.0.1/handler/test", nil)
		apiHandler.ServeHTTP(w, r)
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Equal(t, 0, success)
	})

}

func TestAPIHandler(t *testing.T) {
	success := 0
	failed := 0
	handler := func(w http.ResponseWriter, r *http.Request) {
		success++
	}
	apiHandler := NewAPIHandler(handler, NewAuth(func(r *http.Request) bool {
		return r.FormValue("pass") == "true"
	}, func(w http.ResponseWriter) {
		failed++
		w.WriteHeader(http.StatusBadRequest) // modify status code
	}))
	t.Run("success call", func(t *testing.T) {
		success = 0
		failed = 0
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "http://127.0.0.1/handler/test?pass=true", nil)
		apiHandler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 1, success)
		require.Equal(t, 0, failed)
	})

	t.Run("failed call", func(t *testing.T) {
		success = 0
		failed = 0
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "http://127.0.0.1/handler/test", nil)
		apiHandler.ServeHTTP(w, r)
		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, 0, success)
		require.Equal(t, 1, failed)
	})
}
