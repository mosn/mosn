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
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	rawjson "encoding/json"

	"mosn.io/mosn/pkg/admin/store"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
)

func getEffectiveConfig(port uint32) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/config_dump", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("call admin api failed response status: %d, %s", resp.StatusCode, string(b)))
	}

	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getStats(port uint32) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/stats", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("call admin api failed response status: %d, %s", resp.StatusCode, string(b)))
	}

	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getGlobalStats(port uint32) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/stats_glob", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("call admin api failed response status: %d, %s", resp.StatusCode, string(b)))
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getLoggerLevel() ([]byte, error) {
	url := "http://localhost:8889/api/v1/get_loglevel"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	err = errors.New("get logger info failed")
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
func postUpdateLoggerLevel(port uint32, s string) (string, error) {
	data := strings.NewReader(s)
	url := fmt.Sprintf("http://localhost:%d/api/v1/update_loglevel", port)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("update logger level failed")
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func postToggleLogger(port uint32, logger string, disable bool) (string, error) {
	api := "enable_log" // enable
	if disable {        //disable
		api = "disable_log"
	}
	url := fmt.Sprintf("http://localhost:%d/api/v1/%s", port, api)
	data := strings.NewReader(logger)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("toggle logger failed")
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil

}

func getMosnState(port uint32) (pid int, state store.State, err error) {
	url := fmt.Sprintf("http://localhost:%d/api/v1/states", port)
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, errors.New("get mosn states failed")
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}
	result := strings.Trim(string(b), "\n")
	p := strings.Split(result, "&")
	pidStr := strings.Split(p[0], "=")[1]
	stateStr := strings.Split(p[1], "=")[1]
	pid, _ = strconv.Atoi(pidStr)
	stateInt, _ := strconv.Atoi(stateStr)
	state = store.State(stateInt)
	return pid, state, nil
}

type mockMOSNConfig struct {
	Address string `json:"addr"`
	Port    uint32 `json:"port"`
}

func (m *mockMOSNConfig) GetAdmin() *v2.Admin {
	return &v2.Admin{
		Address: &v2.AddressInfo{
			SocketAddress: v2.SocketAddress{
				Address:   m.Address,
				PortValue: m.Port,
			},
		},
	}
}

func TestDumpConfig(t *testing.T) {
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	mcfg := &v2.MOSNConfig{
		Tracing: v2.TracingConfig{
			Enable: true,
			Driver: "test",
		},
	}
	configmanager.SetMosnConfig(mcfg)
	defer configmanager.Reset()

	time.Sleep(time.Second) //wait server start

	if data, err := getEffectiveConfig(config.Port); err != nil {
		t.Error(err)
	} else {
		if !strings.Contains(data, `tracing":{"enable":true,"driver":"test"}`) {
			t.Errorf("unexpected effectiveConfig: %s\n", data)
		}
	}
}

func TestDumpStats(t *testing.T) {
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start

	stats, _ := metrics.NewMetrics("DumpTest", map[string]string{"lbk1": "lbv1"})
	stats.Counter("ct1").Inc(1)
	stats.Gauge("gg2").Update(3)

	expected, _ := rawjson.MarshalIndent(map[string]map[string]map[string]string{
		"DumpTest": {
			"lbk1.lbv1": {
				"ct1": "1",
				"gg2": "3",
			},
		},
	}, "", "\t")

	if data, err := getStats(config.Port); err != nil {
		t.Error(err)
	} else {
		if data != string(expected) {
			t.Errorf("unexpected stats: %s, expected: %s\n", data, string(expected))
		}
	}

	stats1, _ := metrics.NewMetrics("downstream", map[string]string{"proxy": "global"})
	stats1.Counter("ct1").Inc(1)
	stats1.Gauge("gg2").Update(3)
	expected_string := "ct1:1\ngg2:3\n"
	if data, err := getGlobalStats(config.Port); err != nil {
		t.Error(err)
	} else {
		want := strings.Split(expected_string, "\n")
		got := strings.Split(data, "\n")
		sort.Strings(want)
		sort.Strings(got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected stats: %s\n", data)
		}
	}

	configmanager.Reset()
}

func TestGetLogger(t *testing.T) {
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start

	logName := "/tmp/mosn_admin/test_admin.log"
	_, err := log.GetOrCreateDefaultErrorLogger(logName, log.INFO)
	if err != nil {
		t.Fatal("create logger failed")
	}
	logInfo, err := getLoggerLevel()
	if err != nil {
		t.Fatal(err)
	}
	loggerMap := make(map[string]string)
	json.Unmarshal(logInfo, &loggerMap)
	if loggerMap[logName] != "INFO" {
		t.Errorf("fail to get logger info, %+v", loggerMap)
	}
}

func TestUpdateLogger(t *testing.T) {
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start

	logName := "/tmp/mosn_admin/test_admin.log"
	logger, err := log.GetOrCreateDefaultErrorLogger(logName, log.INFO)
	if err != nil {
		t.Fatal("create logger failed")
	}
	logInfo, err := getLoggerLevel()
	if err != nil {
		t.Fatal(err)
	}
	loggerMap := make(map[string]string)
	json.Unmarshal(logInfo, &loggerMap)
	if loggerMap[logName] != "INFO" {
		t.Errorf("fail to get logger info, %+v", loggerMap)
	}

	// update logger
	postData := `{
		"log_path": "/tmp/mosn_admin/test_admin.log",
		"log_level": "ERROR"
	}`
	if _, err := postUpdateLoggerLevel(config.Port, postData); err != nil {
		t.Fatal(err)
	}
	if logger.GetLogLevel() != log.ERROR {
		t.Errorf("update logger success, but logger level is not expected: %v", logger.GetLogLevel())
	}
	logInfo, err = getLoggerLevel()
	if err != nil {
		t.Fatal(err)
	}
	loggerMap = make(map[string]string)
	json.Unmarshal(logInfo, &loggerMap)
	if loggerMap[logName] != "ERROR" {
		t.Errorf("fail to change logger level, %+v", loggerMap)
	}
}

func TestToggleLogger(t *testing.T) {
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start

	logName := "/tmp/mosn_admin/test_admin_toggler.log"
	os.Remove(logName)
	// Raw Logger
	logger, err := log.GetOrCreateLogger(logName, nil)
	if err != nil {
		t.Fatal("create logger failed")
	}
	// write raw logger, expected write success
	logger.Printf("first")
	// disable logger
	if _, err := postToggleLogger(config.Port, logName, true); err != nil {
		t.Fatal(err)
	}
	// write raw logger, expected write null
	logger.Printf("disable")
	// enable logger
	if _, err := postToggleLogger(config.Port, logName, false); err != nil {
		t.Fatal(err)
	}
	// write raw logger ,expected write success
	logger.Printf("enable")
	time.Sleep(time.Second) // wait flush
	// Verify
	lines, err := readLines(logName)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected write 2 log lines, but got: %d", len(lines))
	}
	if !(lines[0] == "first" && lines[1] == "enable") {
		t.Errorf("log write data is not expected, line1: %s, line2: %s", lines[0], lines[1])
	}

}

func TestGetState(t *testing.T) {
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start

	// init
	pid, state, err := getMosnState(config.Port)
	if err != nil {
		t.Fatal("get mosn states failed")
	}

	// reconfiguring
	store.SetMosnState(store.Passive_Reconfiguring)
	pid2, state2, err := getMosnState(config.Port)
	if err != nil {
		t.Fatal("get mosn states failed")
	}

	// running
	store.SetMosnState(store.Running)
	pid3, state3, err := getMosnState(config.Port)
	if err != nil {
		t.Fatal("get mosn states failed")
	}

	// active reconfiguring
	store.SetMosnState(store.Active_Reconfiguring)
	pid4, state4, err := getMosnState(config.Port)
	if err != nil {
		t.Fatal("get mosn states failed")
	}

	// verify
	curPid := os.Getpid()
	if !(pid == curPid &&
		pid2 == curPid &&
		pid3 == curPid &&
		pid4 == curPid) {
		t.Error("mosn pid is not expected", pid, pid2, pid3, pid4)
	}
	if !(state == store.Init &&
		state2 == store.Passive_Reconfiguring &&
		state3 == store.Running &&
		state4 == store.Active_Reconfiguring) {
		t.Error("mosn state is not expected", state, state2, state3, state4)
	}
}

func TestRegisterNewAPI(t *testing.T) {
	// register api before start
	newAPI := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("new api"))
	}
	pattern := "/api/new/test"
	RegisterAdminHandleFunc(pattern, newAPI)
	//
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start
	url := fmt.Sprintf("http://localhost:%d%s", config.Port, pattern)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestRegisterAPIHandler(t *testing.T) {
	handler := NewAPIHandler(func(w http.ResponseWriter, r *http.Request) {
	}, NewAuth(func(r *http.Request) bool {
		return r.FormValue("pass") == "true"
	}, nil))
	pattern := "/api/handler/auth"
	RegisterAdminHandler(pattern, handler)
	//
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start
	t.Run("request success", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%d%s?pass=true", config.Port, pattern)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: %d", resp.StatusCode)
		}
	})
	t.Run("request failed", func(t *testing.T) {
		url := fmt.Sprintf("http://localhost:%d%s", config.Port, pattern)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("unexpected status code: %d", resp.StatusCode)
		}
	})
}

func TestHelpAPI(t *testing.T) {
	// reset
	apiHandlerStore = map[string]*APIHandler{
		"/api/v1/test_deleted":    NewAPIHandler(ConfigDump), // will be deleted
		"/api/v1/config_dump":     NewAPIHandler(ConfigDump),
		"/api/v1/stats":           NewAPIHandler(StatsDump),
		"/api/v1/update_loglevel": NewAPIHandler(UpdateLogLevel),
		"/api/v1/get_loglevel":    NewAPIHandler(GetLoggerInfo),
		"/api/v1/enable_log":      NewAPIHandler(EnableLogger),
		"/api/v1/disable_log":     NewAPIHandler(DisableLogger),
		"/api/v1/states":          NewAPIHandler(GetState),
		"/":                       NewAPIHandler(Help),
	}
	time.Sleep(time.Second)
	server := Server{}
	config := &mockMOSNConfig{
		Address: "127.0.0.1",
		Port:    8889,
	}
	// delete
	DeleteRegisteredAdminHandler("/api/v1/test_deleted")
	server.Start(config)
	store.StartService(nil)
	defer store.StopService()

	time.Sleep(time.Second) //wait server start
	query := func(t *testing.T, addr string) string {
		resp, err := http.Get("http://127.0.0.1:8889" + addr)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("query help message, returns: %d", resp.StatusCode)
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	// root, and no registered addr should returns help
	for _, addr := range []string{"/", "/not_exists"} {
		s := query(t, addr)
		s = strings.TrimSuffix(s, "\n")
		apis := strings.Split(s, "\n")[1:] // the first line is "supported APIs:"
		if len(apis) != 7 {                // exclued "/"
			t.Errorf("apis count is not expected: %v, length is %d", apis, len(apis))
		}
	}

}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
