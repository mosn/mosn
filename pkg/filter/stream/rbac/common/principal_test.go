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

package common

import (
	"testing"
)

func TestPrincipalAny(t *testing.T) {
	engine, _, err := genRoleBasedAccessControlEngine("./test_conf/deny-all.json")
	if err != nil {
		t.Error("TestPrincipalAny failed")
		return
	}

	allowed, _ := engine.Allowed(nil, nil)
	if allowed {
		t.Error("TestPrincipalAny failed")
		return
	}
}

func TestPrincipalOrIds(t *testing.T) {
	engine, _, err := genRoleBasedAccessControlEngine("./test_conf/principal-or.json")
	if err != nil {
		t.Error("TestPrincipalOrIds failed")
		return
	}

	allowed, _ := engine.Allowed(nil, nil)
	if allowed {
		t.Error("TestPrincipalOrIds failed")
		return
	}

	engine, _, err = genRoleBasedAccessControlEngine("./test_conf/principal-and.json")
	if err != nil {
		t.Error("TestPrincipalOrIds failed")
		return
	}

	allowed, _ = engine.Allowed(nil, nil)
	if !allowed {
		t.Error("TestPrincipalOrIds failed")
		return
	}
}

func TestPrincipalAndIds(t *testing.T) {
	engine, _, err := genRoleBasedAccessControlEngine("./test_conf/principal-and.json")
	if err != nil {
		t.Error("TestPrincipalAndIds failed")
		return
	}

	allowed, _ := engine.Allowed(nil, nil)
	if !allowed {
		t.Error("TestPrincipalAndIds failed")
		return
	}

	engine, _, err = genRoleBasedAccessControlEngine("./test_conf/principal-or.json")
	if err != nil {
		t.Error("TestPrincipalAndIds failed")
		return
	}

	allowed, _ = engine.Allowed(nil, nil)
	if allowed {
		t.Error("TestPrincipalAndIds failed")
		return
	}
}

func TestPrincipalSourceIp(t *testing.T) {
	engine, _, err := genRoleBasedAccessControlEngine("./test_conf/principal-src-ip.json")
	if err != nil {
		t.Error("TestPrincipalSourceIp failed")
		return
	}

	cb := &mockStreamReceiverFilterHandler{
		conn: &mockConn{
			remoteAddr: &mockAddr{
				IP:   "1.2.3.4",
				Port: 2345,
			},
		},
	}

	allowed, _ := engine.Allowed(cb, nil)
	if allowed {
		t.Error("TestPrincipalSourceIp failed")
		return
	}

	cb.conn.remoteAddr.IP = "1.2.3.5"
	allowed, _ = engine.Allowed(cb, nil)
	if !allowed {
		t.Error("TestPrincipalSourceIp failed")
		return
	}
}

func TestPrincipalHeader(t *testing.T) {
	// Present
	engine, _, err := genRoleBasedAccessControlEngine("./test_conf/principal-headers-present.json")
	if err != nil {
		t.Error("TestPrincipalHeader failed")
		return
	}

	headers := &mockHeaderMap{
		headers: map[string]string{
			"X-Custom-Header": "123",
		},
	}

	allowed, _ := engine.Allowed(nil, headers)
	if allowed {
		t.Error("TestPrincipalHeader failed")
		return
	}

	delete(headers.headers, "X-Custom-Header")

	allowed, _ = engine.Allowed(nil, headers)
	if !allowed {
		t.Error("TestPrincipalHeader failed")
		return
	}

	// Header Value
	engine, _, err = genRoleBasedAccessControlEngine("./test_conf/principal-headers-value.json")
	if err != nil {
		t.Error("TestPrincipalHeader failed")
		return
	}

	headers.headers["X-Mosn-Path"] = "/control-api/resources"
	allowed, _ = engine.Allowed(nil, headers)
	if allowed {
		t.Error("TestPrincipalHeader failed")
		return
	}

	headers.headers["X-Mosn-Path"] = "/sources/test.java"
	allowed, _ = engine.Allowed(nil, headers)
	if allowed {
		t.Error("TestPrincipalHeader failed")
		return
	}

	headers.headers["X-Mosn-Path"] = "/deny/custom"
	allowed, _ = engine.Allowed(nil, headers)
	if allowed {
		t.Error("TestPrincipalHeader failed")
		return
	}

	delete(headers.headers, "X-Mosn-Path")
	headers.headers["X-Mosn-Method"] = "HEAD"
	allowed, _ = engine.Allowed(nil, headers)
	if allowed {
		t.Error("TestPrincipalHeader failed")
		return
	}
}
