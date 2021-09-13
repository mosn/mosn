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

package plugin

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mosn.io/mosn/pkg/plugin/proto"
)

func testRegister(name string) (*Client, error) {
	return Register(name, &Config{Args: []string{"-test.run=TestHelperProcess"}})
}

func TestPluginClient(t *testing.T) {
	// init log base
	InitPlugin("/tmp")
	if GetLogPath() != "/tmp/" {
		t.Fatalf("TestNewPluginClient error, GetLogPath error")
	}
	InitPlugin("/tmp/1/")
	if GetLogPath() != "/tmp/1/" {
		t.Fatalf("TestNewPluginClient error, GetLogPath error")
	}
	// test admin api
	server, err := NewHttp("127.0.0.1:34567")
	if err != nil {
		t.Fatalf("TestNewPluginClient error, NewHttp error :%v", err)
	}

	go server.ListenAndServe()
	defer server.Close()

	testname := filepath.Base(os.Args[0])
	client, err := testRegister(testname)
	if err != nil {
		t.Fatalf("TestNewPluginClient error:%v", err)
	}
	defer client.disable()
	client1, err := testRegister(testname)
	if client != client1 {
		t.Fatalf("TestNewPluginClient error: client should equal client1")
	}

	if err := client.Check(); err != nil {
		t.Fatalf("TestNewPluginClient error:%v", err)
	}

	// test request
	header := make(map[string]string)
	header["a"] = "a"
	body := []byte("hello")
	request := &proto.Request{Header: header, Body: body}
	response, err := client.Call(request, 0)
	if err != nil {
		t.Fatalf("TestNewPluginClient error:%v", err)
	}
	if response.Header["a"] != "b" {
		t.Fatalf("TestNewPluginClient error, respsone.Header should be b")
	}
	if string(response.Body) != "world" {
		t.Fatalf("TestNewPluginClient error, respsone.Body should be world")
	}
	if response.Status != 1 {
		t.Fatalf("TestNewPluginClient error, respsone.Status should be 1")
	}
	t.Logf("TestNewPluginClient request:%v, response:%v", request, response)

	// test timeout
	response, err = client.Call(request, 100*time.Millisecond)
	if err == nil {
		t.Fatalf("TestNewPluginClient error, request should be timeout")
	}
	t.Logf("TestNewPluginClient request timeout :%v", err)

	// test CheckPluginStatus
	status, err := CheckPluginStatus(testname)
	if err != nil {
		t.Fatalf("TestNewPluginClient error, CheckPluginStatus error %v", err)
	}
	if status != "name:"+testname+",enable:true,on:true" {
		t.Fatalf("TestNewPluginClient error, CheckPluginStatus error: %s", status)
	}
	res, err := http.Get("http://localhost:34567/plugin?status=" + testname)
	if res == nil || res.StatusCode != http.StatusOK {
		t.Fatalf("TestNewPluginClient error, /plugin?status=all error: %v", err)
	}

	// test ClosePlugin
	res, err = http.Get("http://localhost:34567/plugin?disable=" + testname)
	if res == nil || res.StatusCode != http.StatusOK {
		t.Fatalf("TestNewPluginClient error, /plugin?status=all error: %v", err)
	}
	status, err = CheckPluginStatus(testname)
	if err != nil {
		t.Fatalf("TestNewPluginClient error, CheckPluginStatus error %v", err)
	}
	if status != "name:"+testname+",enable:false,on:false" {
		t.Fatalf("TestNewPluginClient error, CheckPluginStatus error: %s", status)
	}

	// test failed
	response, err = client.Call(request, 100*time.Millisecond)
	if err == nil {
		t.Fatalf("TestNewPluginClient error, request should be failed")
	}

	// test OpenPlugin
	res, err = http.Get("http://localhost:34567/plugin?enable=" + testname)
	if res == nil || res.StatusCode != http.StatusOK {
		t.Fatalf("TestNewPluginClient error, /plugin?status=all error: %v", err)
	}
	status, err = CheckPluginStatus(testname)
	if err != nil {
		t.Fatalf("TestNewPluginClient error, CheckPluginStatus error %v", err)
	}
	if status != "name:"+testname+",enable:true,on:true" {
		t.Fatalf("TestNewPluginClient error, CheckPluginStatus error: %s", status)
	}
	response, err = client.Call(request, 0)
	if err != nil {
		t.Fatalf("TestNewPluginClient error, request error:%v", err)
	}

	res, err = http.Get("http://localhost:34567/plugin?status=all")
	if res == nil || res.StatusCode != http.StatusOK {
		t.Fatalf("TestNewPluginClient error, /plugin?status=all error: %v", err)
	}

	// test help
	res, err = http.Get("http://localhost:34567/")
	if res == nil || res.StatusCode == http.StatusOK {
		t.Fatalf("TestNewPluginClient error, /plugin?status=all error: %v", err)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("TestNewPluginClient error, admin help error: %v", err)
	}
	if !strings.Contains(string(b), "Usage:") {
		t.Fatalf("TestNewPluginClient error, admin help error: %v", b)
	}
}

// This is not a real test. This is just a helper process kicked off by tests.
func TestHelperProcess(*testing.T) {
	if os.Getenv("MOSN_PROCS") == "" {
		return
	}
	defer os.Exit(0)

	Serve(new(testService))
}

type testService struct{}

func (s *testService) Call(request *proto.Request) (*proto.Response, error) {
	header := request.GetHeader()
	body := request.GetBody()
	if header["a"] == "a" {
		header["a"] = "b"
	}
	if string(body) == "hello" {
		body = []byte("world")
	}

	response := new(proto.Response)
	response.Header = header
	response.Body = body
	response.Status = 1

	// test timeout
	time.Sleep(1000 * time.Millisecond)

	return response, nil
}
