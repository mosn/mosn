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

package store

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestService(t *testing.T) {
	var b bool
	srv := &http.Server{
		Addr:    "127.0.0.1:13475",
		Handler: nil,
	}
	AddService(
		srv,
		"test",
		func() {
			b = true
		},
		func() {
			b = false
		})
	StartService(nil)
	if !b {
		t.Errorf("TestService init func error")
	}

	var b1 bool
	srv1 := &http.Server{
		Addr:    "127.0.0.1:13476",
		Handler: nil,
	}
	AddService(
		srv1,
		"test1",
		func() {
			b1 = true
		},
		func() {
			b1 = false
		})

	StartService(nil)
	if !b1 {
		t.Errorf("TestService init func error")
	}

	files, err := ListServiceListenersFile()
	if len(files) != 2 {
		t.Errorf("TestService ListServiceListenersFile() error: %v", err)
	}
	StopService()
	if b {
		t.Errorf("TestService exit func error")
	}
	if b1 {
		t.Errorf("TestService exit func error")
	}
}

func TestServiceInherit(t *testing.T) {
	mux1 := http.NewServeMux()
	mux1.HandleFunc("/", handler1)
	s := &http.Server{
		Addr:    "127.0.0.1:13477",
		Handler: mux1,
	}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		t.Errorf("TestService net.Listern error: %v", err)
	}
	go s.Serve(ln)

	time.Sleep(time.Second)
	rsp, err := http.Get("http://127.0.0.1:13477/")
	if rsp == nil || rsp.StatusCode != 200 {
		t.Errorf("TestService http.Get error: %v", rsp)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("/", handler2)
	srv := &http.Server{
		Addr:    "127.0.0.1:13477",
		Handler: mux2,
	}
	AddService(srv, "test", nil, nil)

	l := ln.(*net.TCPListener)
	f, _ := l.File()
	ll, _ := net.FileListener(f)
	f.Close()

	if err := StartService([]net.Listener{ll}); err != nil {
		t.Errorf("TestService StartService error: %v", err)
	}
	// close old server
	s.Shutdown(context.Background())

	time.Sleep(time.Second)
	rsp, err = http.Get("http://127.0.0.1:13477/")
	if rsp == nil || rsp.StatusCode != 201 {
		t.Errorf("TestService http.Get error: %v", rsp)
	}

	StopService()

	time.Sleep(time.Second)
	rsp, err = http.Get("http://127.0.0.1:13477/")
	if err == nil {
		t.Errorf("TestService http.Get should failed")
	}

}

func handler1(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func handler2(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(201)
}
