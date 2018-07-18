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

package log

//
//import (
//	"testing"
//	"github.com/alipay/sofa-mosn/pkg/network"
//	"net"
//	"time"
//	"fmt"
//)
//
//func BenchmarkAccessLog(b *testing.B) {
//	InitDefaultLogger("", INFO)
//	// ~ replace the path if needed
//	accessLog, err := NewAccessLog("/tmp/mosn_bench/benchmark_access.log", nil, "")
//
//	if err != nil {
//		fmt.Errorf(err.Error())
//	}
//	reqHeaders := map[string]string{
//		"service": "test",
//	}
//
//	respHeaders := map[string]string{
//		"Server": "MOSN",
//	}
//
//	requestInfo := network.NewRequestInfoWithPort("Http1")
//	requestInfo.SetRequestReceivedDuration(time.Now())
//	requestInfo.SetResponseReceivedDuration(time.Now().Add(time.Second * 2))
//	requestInfo.SetBytesSent(2048)
//	requestInfo.SetBytesReceived(2048)
//
//	requestInfo.SetResponseFlag(0)
//	requestInfo.SetUpstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 23456, ""})
//	requestInfo.SetDownstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 12200, ""})
//	requestInfo.SetDownstreamRemoteAddress(&net.TCPAddr{[]byte("127.0.0.2"), 53242, ""})
//	requestInfo.OnUpstreamHostSelected(nil)
//
//	for n := 0; n < b.N; n++ {
//		accessLog.Log(reqHeaders, respHeaders, requestInfo)
//	}
//}
