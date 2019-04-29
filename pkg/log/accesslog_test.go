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

import (
	"fmt"
	"io/ioutil"
	"net"
	"runtime"
	"testing"
	"time"

	"os"
	"regexp"

	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func TestAccessLog(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	format := "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %BytesSent%" + " " +
		"%BytesReceived% %Protocol% %ResponseCode% %Duration% %ResponseFlag% %ResponseCode% %UpstreamLocalAddress%" + " " +
		"%DownstreamLocalAddress% %DownstreamRemoteAddress% %UpstreamHostSelected%"
	logName := "/tmp/mosn_bench/benchmark_access.log"
	os.Remove(logName)
	accessLog, err := NewAccessLog(logName, nil, format)

	if err != nil {
		t.Errorf(err.Error())
	}
	reqHeaders := map[string]string{
		"service": "test",
	}

	respHeaders := map[string]string{
		"Server": "MOSN",
	}
	requestInfo := newRequestInfo()
	requestInfo.SetRequestReceivedDuration(time.Now())
	requestInfo.SetResponseReceivedDuration(time.Now().Add(time.Second * 2))
	requestInfo.SetBytesSent(2048)
	requestInfo.SetBytesReceived(2048)

	requestInfo.SetResponseFlag(0)
	requestInfo.SetUpstreamLocalAddress(&net.TCPAddr{net.ParseIP("127.0.0.1"), 23456, ""})
	requestInfo.SetDownstreamLocalAddress(&net.TCPAddr{net.ParseIP("2001:db8::68"), 12200, ""})
	requestInfo.SetDownstreamRemoteAddress(&net.TCPAddr{net.ParseIP("127.0.0.1"), 53242, ""})
	requestInfo.OnUpstreamHostSelected(nil)

	accessLog.Log(protocol.CommonHeader(reqHeaders), protocol.CommonHeader(respHeaders), requestInfo)
	l := "2018/12/14 18:08:33.054 1.329µs 2.00000227s 2048 2048 - 0 126.868µs false 0 127.0.0.1:23456 [2001:db8::68]:12200 127.0.0.1:53242 -\n"
	time.Sleep(2 * time.Second)
	f, _ := os.Open(logName)
	b := make([]byte, 1024)
	_, err = f.Read(b)
	f.Close()
	if err != nil {
		t.Errorf("test accesslog error")
	}
	ok, err := regexp.Match("\\d\\d\\d\\d/\\d\\d/\\d\\d .* .* .* 2048 2048 \\- 0 .* false 0 127.0.0.1:23456 \\[2001:db8::68\\]:12200 127.0.0.1:53242 \\-\n", []byte(l))

	if !ok {
		t.Errorf("test accesslog error %v", err)
	}
}

func TestAccessLogStartTime(t *testing.T) {
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond)
		now := time.Now()
		t.Log(now.Format("2006/01/02 15:04:05.999"))
		t.Log(now.Format("2006/01/02 15:04:05.000"))
	}
}

func TestAccessLogDisable(t *testing.T) {
	DefaultDisableAccessLog = true
	format := "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %BytesSent%" + " " +
		"%BytesReceived% %Protocol% %ResponseCode% %Duration% %ResponseFlag% %ResponseCode% %UpstreamLocalAddress%" + " " +
		"%DownstreamLocalAddress% %DownstreamRemoteAddress% %UpstreamHostSelected%"
	logName := "/tmp/mosn_accesslog/disbale_access.log"
	os.Remove(logName)
	accessLog, err := NewAccessLog(logName, nil, format)
	if err != nil {
		t.Fatal(err)
	}
	reqHeaders := map[string]string{
		"service": "test",
	}

	respHeaders := map[string]string{
		"Server": "MOSN",
	}
	requestInfo := newRequestInfo()
	requestInfo.SetRequestReceivedDuration(time.Now())
	requestInfo.SetResponseReceivedDuration(time.Now().Add(time.Second * 2))
	requestInfo.SetBytesSent(2048)
	requestInfo.SetBytesReceived(2048)

	requestInfo.SetResponseFlag(0)
	requestInfo.SetUpstreamLocalAddress(&net.TCPAddr{net.ParseIP("127.0.0.1"), 23456, ""})
	requestInfo.SetDownstreamLocalAddress(&net.TCPAddr{net.ParseIP("2001:db8::68"), 12200, ""})
	requestInfo.SetDownstreamRemoteAddress(&net.TCPAddr{net.ParseIP("127.0.0.1"), 53242, ""})
	requestInfo.OnUpstreamHostSelected(nil)
	// try write disbale access log nothing happened
	accessLog.Log(protocol.CommonHeader(reqHeaders), protocol.CommonHeader(respHeaders), requestInfo)
	time.Sleep(time.Second)
	if b, err := ioutil.ReadFile(logName); err != nil || len(b) > 0 {
		t.Fatalf("verify log file failed, data len: %d, error: %v", len(b), err)
	}
	// enable access log
	if !ToggleLogger(logName, false) {
		t.Fatal("enable access log failed")
	}
	// retry, write success
	accessLog.Log(protocol.CommonHeader(reqHeaders), protocol.CommonHeader(respHeaders), requestInfo)
	time.Sleep(time.Second)
	if b, err := ioutil.ReadFile(logName); err != nil || len(b) == 0 {
		t.Fatalf("verify log file failed, data len: %d, error: %v", len(b), err)
	}
}

func TestAccessLogManage(t *testing.T) {
	defer CloseAll()
	DefaultDisableAccessLog = false
	format := "%StartTime% %ResponseFlag%"
	var logs []types.AccessLog
	for i := 0; i < 100; i++ {
		logName := fmt.Sprintf("/tmp/accesslog.%d.log", i)
		lg, err := NewAccessLog(logName, nil, format)
		if err != nil {
			t.Fatal(err)
		}
		logs = append(logs, lg)
	}
	DisableAllAccessLog()
	// new access log is auto disabled
	for i := 200; i < 300; i++ {
		logName := fmt.Sprintf("/tmp/accesslog.%d.log", i)
		lg, err := NewAccessLog(logName, nil, format)
		if err != nil {
			t.Fatal(err)
		}
		logs = append(logs, lg)
	}
	// verify
	// all accesslog is disabled
	for _, lg := range logs {
		alg := lg.(*accesslog)
		if !alg.logger.disable {
			t.Fatal("some access log is enabled")
		}
	}
}

func BenchmarkAccessLog(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	InitDefaultLogger("", INFO)
	// ~ replace the path if needed
	accessLog, err := NewAccessLog("/tmp/mosn_bench/benchmark_access.log", nil, "")

	if err != nil {
		fmt.Errorf(err.Error())
	}
	reqHeaders := map[string]string{
		"service": "test",
	}

	respHeaders := map[string]string{
		"Server": "MOSN",
	}

	requestInfo := newRequestInfo()
	requestInfo.SetRequestReceivedDuration(time.Now())
	requestInfo.SetResponseReceivedDuration(time.Now().Add(time.Second * 2))
	requestInfo.SetBytesSent(2048)
	requestInfo.SetBytesReceived(2048)

	requestInfo.SetResponseFlag(0)
	requestInfo.SetUpstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 23456, ""})
	requestInfo.SetDownstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 12200, ""})
	requestInfo.SetDownstreamRemoteAddress(&net.TCPAddr{[]byte("127.0.0.2"), 53242, ""})
	requestInfo.OnUpstreamHostSelected(nil)

	for n := 0; n < b.N; n++ {
		accessLog.Log(protocol.CommonHeader(reqHeaders), protocol.CommonHeader(respHeaders), requestInfo)
	}
}

func BenchmarkAccessLogParallel(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	InitDefaultLogger("", INFO)
	// ~ replace the path if needed
	accessLog, err := NewAccessLog("/tmp/mosn_bench/benchmark_access.log", nil, "")

	if err != nil {
		b.Errorf(err.Error())
	}
	reqHeaders := map[string]string{
		"service": "test",
	}

	respHeaders := map[string]string{
		"Server": "MOSN",
	}

	requestInfo := newRequestInfo()
	requestInfo.SetRequestReceivedDuration(time.Now())
	requestInfo.SetResponseReceivedDuration(time.Now().Add(time.Second * 2))
	requestInfo.SetBytesSent(2048)
	requestInfo.SetBytesReceived(2048)

	requestInfo.SetResponseFlag(0)
	requestInfo.SetUpstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 23456, ""})
	requestInfo.SetDownstreamLocalAddress(&net.TCPAddr{[]byte("127.0.0.1"), 12200, ""})
	requestInfo.SetDownstreamRemoteAddress(&net.TCPAddr{[]byte("127.0.0.2"), 53242, ""})
	requestInfo.OnUpstreamHostSelected(nil)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			accessLog.Log(protocol.CommonHeader(reqHeaders), protocol.CommonHeader(respHeaders), requestInfo)
		}
	})
}

// mock_requestInfo
type mock_requestInfo struct {
	protocol                 types.Protocol
	startTime                time.Time
	responseFlag             types.ResponseFlag
	upstreamHost             types.HostInfo
	requestReceivedDuration  time.Duration
	responseReceivedDuration time.Duration
	requestFinishedDuration  time.Duration
	bytesSent                uint64
	bytesReceived            uint64
	responseCode             int
	localAddress             net.Addr
	downstreamLocalAddress   net.Addr
	downstreamRemoteAddress  net.Addr
	isHealthCheckRequest     bool
	routerRule               types.RouteRule
}

// NewrequestInfo
func newRequestInfo() types.RequestInfo {
	return &mock_requestInfo{
		startTime: time.Now(),
	}
}

func (r *mock_requestInfo) StartTime() time.Time {
	return r.startTime
}

func (r *mock_requestInfo) SetStartTime() {
	r.startTime = time.Now()
}

func (r *mock_requestInfo) RequestReceivedDuration() time.Duration {
	return r.requestReceivedDuration
}

func (r *mock_requestInfo) SetRequestReceivedDuration(t time.Time) {
	r.requestReceivedDuration = t.Sub(r.startTime)
}

func (r *mock_requestInfo) ResponseReceivedDuration() time.Duration {
	return r.responseReceivedDuration
}

func (r *mock_requestInfo) SetResponseReceivedDuration(t time.Time) {
	r.responseReceivedDuration = t.Sub(r.startTime)
}

func (r *mock_requestInfo) RequestFinishedDuration() time.Duration {
	return r.requestFinishedDuration
}

func (r *mock_requestInfo) SetRequestFinishedDuration(t time.Time) {
	r.requestFinishedDuration = t.Sub(r.startTime)
}

func (r *mock_requestInfo) BytesSent() uint64 {
	return r.bytesSent
}

func (r *mock_requestInfo) SetBytesSent(bytesSent uint64) {
	r.bytesSent = bytesSent
}

func (r *mock_requestInfo) BytesReceived() uint64 {
	return r.bytesReceived
}

func (r *mock_requestInfo) SetBytesReceived(bytesReceived uint64) {
	r.bytesReceived = bytesReceived
}

func (r *mock_requestInfo) Protocol() types.Protocol {
	return r.protocol
}

func (r *mock_requestInfo) ResponseCode() int {
	return r.responseCode
}

func (r *mock_requestInfo) SetResponseCode(code int) {
	r.responseCode = code
}

func (r *mock_requestInfo) Duration() time.Duration {
	return time.Now().Sub(r.startTime)
}

func (r *mock_requestInfo) GetResponseFlag(flag types.ResponseFlag) bool {
	return r.responseFlag&flag != 0
}

func (r *mock_requestInfo) SetResponseFlag(flag types.ResponseFlag) {
	r.responseFlag |= flag
}

func (r *mock_requestInfo) UpstreamHost() types.HostInfo {
	return r.upstreamHost
}

func (r *mock_requestInfo) OnUpstreamHostSelected(host types.HostInfo) {
	r.upstreamHost = host
}

func (r *mock_requestInfo) UpstreamLocalAddress() net.Addr {
	return r.localAddress
}

func (r *mock_requestInfo) SetUpstreamLocalAddress(addr net.Addr) {
	r.localAddress = addr
}

func (r *mock_requestInfo) IsHealthCheck() bool {
	return r.isHealthCheckRequest
}

func (r *mock_requestInfo) SetHealthCheck(isHc bool) {
	r.isHealthCheckRequest = isHc
}

func (r *mock_requestInfo) DownstreamLocalAddress() net.Addr {
	return r.downstreamLocalAddress
}

func (r *mock_requestInfo) SetDownstreamLocalAddress(addr net.Addr) {
	r.downstreamLocalAddress = addr
}

func (r *mock_requestInfo) DownstreamRemoteAddress() net.Addr {
	return r.downstreamRemoteAddress
}

func (r *mock_requestInfo) SetDownstreamRemoteAddress(addr net.Addr) {
	r.downstreamRemoteAddress = addr
}

func (r *mock_requestInfo) RouteEntry() types.RouteRule {
	return r.routerRule
}

func (r *mock_requestInfo) SetRouteEntry(routerRule types.RouteRule) {
	r.routerRule = routerRule
}
