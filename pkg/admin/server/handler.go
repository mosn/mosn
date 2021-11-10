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

// Auth contains two parts: check function and failed function.
// if check function returns false, the Auth.Check will returns false and
// call failed function. if failed function is nil, use default instead.
// default function will write a http forbidden without body
type Auth struct {
	checkAction  func(*http.Request) bool
	failedAction func(http.ResponseWriter)
}

func NewAuth(check func(*http.Request) bool, failed func(http.ResponseWriter)) *Auth {
	if failed == nil {
		failed = defaultFailedFunc
	}
	return &Auth{
		checkAction:  check,
		failedAction: failed,
	}
}

func (a Auth) Check(w http.ResponseWriter, r *http.Request) bool {
	pass := true
	if a.checkAction != nil {
		pass = a.checkAction(r)
		if !pass {
			a.failedAction(w)
		}
	}
	return pass
}

// APIHandler is a wrapper of http.Handler, which contains auth options
type APIHandler struct {
	// the real handler function
	handler func(http.ResponseWriter, *http.Request)
	// chains for auth action, different auth can be combined for different handler.
	// one of the auths check returns false, the handler will not be called.
	auths []*Auth
}

var _ http.Handler = (*APIHandler)(nil)

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// default check is ok, if auths is empty, pass
	for _, auth := range h.auths {
		if !auth.Check(w, r) {
			log.DefaultLogger.Errorf("[admin] request %+v failed by auth.", r)
			return
		}
	}
	h.handler(w, r)
}

func defaultFailedFunc(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
}

func NewAPIHandler(handler func(http.ResponseWriter, *http.Request), auths ...*Auth) *APIHandler {
	return &APIHandler{
		handler: handler,
		auths:   auths,
	}
}
