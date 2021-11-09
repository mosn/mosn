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

	"mosn.io/mosn/pkg/log"
)

// APIHandler is a wrapper of http.Handler, which contains auth options
type APIHandler struct {
	// the real handler function
	handler func(http.ResponseWriter, *http.Request)
	// chains for auth, one of the auth returns false, the request will be failed.
	auths []func(*http.Request) bool
	// write repsonse when the request failed by auth function
	// default is 403 Forbidden without body
	failed func(http.ResponseWriter)
}

var _ http.Handler = (*APIHandler)(nil)

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// default check is ok, if auths is empty, pass
	for _, auth := range h.auths {
		if !auth(r) {
			log.DefaultLogger.Errorf("[admin] request %+v failed by auth.", r)
			h.failed(w)
			return
		}
	}
	h.handler(w, r)
}

func defaultFailedFunc(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
}

func NewAPIHandler(handler func(http.ResponseWriter, *http.Request), failed func(http.ResponseWriter), auths ...func(*http.Request) bool) *APIHandler {
	if failed == nil {
		failed = defaultFailedFunc
	}
	return &APIHandler{
		handler: handler,
		auths:   auths,
		failed:  failed,
	}
}
