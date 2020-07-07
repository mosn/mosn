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

package network

import (
	"io/ioutil"
	"testing"

	v2 "mosn.io/mosn/pkg/config/v2"
)

func TestDnsResolve(t *testing.T) {
	dnsResolver := NewDnsResolverFromFile("", "")
	if dnsResolver == nil {
		t.Error("create dns resolver failed")

		return
	}
	res := dnsResolver.DnsResolve("www.baidu.com", "V4_ONLY")
	if res == nil {
		t.Skip("skipping test, maybe network timeout caused resolve failed.")
	}

	resolveConfig := "options timeout:1 attempts:1\nnameserver 114.114.114.114\nnameserver 8.8.8.8\nnameserver 8.8.4.4\nnameserver 223.5.5.5"
	var fileName = "/tmp/resolve.conf"
	if err := ioutil.WriteFile(fileName, []byte(resolveConfig), 0644); err != nil {
		t.Fatal(err)
	}
	dnsResolver = NewDnsResolverFromFile(fileName, "53")
	res = dnsResolver.DnsResolve("www.baidu.com", "V4Only")
	if res == nil {
		t.Error("resolve dns failed")
	}

	config := &v2.DnsResolverConfig{
		Servers:  []string{"114.114.114.114", "8.8.8.8"},
		Search:   nil,
		Port:     "53",
		Ndots:    0,
		Timeout:  2,
		Attempts: 0,
	}

	dnsResolver = NewDnsResolver(config)
	for _, server := range dnsResolver.clientConfig.Servers {
		if server != "114.114.114.114" && server != "8.8.8.8" {
			t.Error("new dns resolver not from config")
		}
	}
}
