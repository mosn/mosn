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
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTCPDial(t *testing.T) {
	s := httptest.NewServer(nil)
	addr := strings.Split(s.URL, "http://")[1]
	host := &mockHost{
		addr: addr,
	}
	dialfactory := &TCPDialSessionFactory{}
	session := dialfactory.NewSession(nil, host)
	ctx1, cancel1 := context.WithTimeout(context.TODO(), time.Second)
	defer cancel1()
	if !session.CheckHealth(ctx1) {
		t.Error("tcp dial check health failed")
	}
	s.Close()
	ctx2, cancel2 := context.WithTimeout(context.TODO(), time.Second)
	defer cancel2()
	if session.CheckHealth(ctx2) {
		t.Error("tcp dial a closed server, but returns ok")
	}
}
