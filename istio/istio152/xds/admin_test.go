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

package xds

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	envoy_admin_v2alpha "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/mosn/istio/istio152/xds/conv"
	"mosn.io/mosn/pkg/admin/store"
)

func TestGetState(t *testing.T) {
	ads := &AdsConfig{
		converter: conv.NewConverter(),
	}
	getMosnStateForIstio := func() (state envoy_admin_v2alpha.ServerInfo_State, err error) {
		w := httptest.NewRecorder()
		serverInfoForIstio(w, nil)

		if w.Code != http.StatusOK {
			return 0, errors.New("get mosn states for istio failed")
		}
		serverInfo := envoy_admin_v2alpha.ServerInfo{}
		if err := jsonpb.Unmarshal(w.Body, &serverInfo); err != nil {
			return 0, err
		}
		return serverInfo.GetState(), nil
	}
	getStatsForIstio := func() (statsInfo string, err error) {
		w := httptest.NewRecorder()
		ads.statsForIstio(w, nil)

		if w.Code != http.StatusOK {
			return "", errors.New("get mosn stats for istio failed")
		}
		b, err := ioutil.ReadAll(w.Body)
		if err != nil {
			return "", err
		}

		return string(b), nil
	}

	// init
	stateForIstio, err := getMosnStateForIstio()
	if err != nil {
		t.Fatalf("get mosn states for istio failed: %v", err)
	}
	stats, err := getStatsForIstio()
	if err != nil {
		t.Fatalf("get mosn stats for istio failed: %v", err)
	}

	// reconfiguring
	store.SetMosnState(store.Passive_Reconfiguring)
	stateForIstio2, err := getMosnStateForIstio()
	if err != nil {
		t.Fatal("get mosn states for istio failed")
	}
	stats2, err := getStatsForIstio()
	if err != nil {
		t.Fatal("get mosn stats for istio failed")
	}

	// running
	store.SetMosnState(store.Running)
	stateForIstio3, err := getMosnStateForIstio()
	if err != nil {
		t.Fatal("get mosn states for istio failed")
	}
	stats3, err := getStatsForIstio()
	if err != nil {
		t.Fatal("get mosn stats for istio failed")
	}

	// active reconfiguring
	store.SetMosnState(store.Active_Reconfiguring)
	stateForIstio4, err := getMosnStateForIstio()
	if err != nil {
		t.Fatal("get mosn states for istio failed")
	}
	stats4, err := getStatsForIstio()
	if err != nil {
		t.Fatal("get mosn stats for istio failed")
	}

	// verify
	if !(stateForIstio == envoy_admin_v2alpha.ServerInfo_INITIALIZING &&
		stateForIstio2 == envoy_admin_v2alpha.ServerInfo_DRAINING &&
		stateForIstio3 == envoy_admin_v2alpha.ServerInfo_LIVE &&
		stateForIstio4 == envoy_admin_v2alpha.ServerInfo_PRE_INITIALIZING) {
		t.Error("mosn state for istio is not expected", stateForIstio, stateForIstio2, stateForIstio3, stateForIstio4)
	}
	prefix := fmt.Sprintf("%s: ", SERVER_STATE)
	stateMatched, err := regexp.MatchString(fmt.Sprintf("%s%d", prefix, envoy_admin_v2alpha.ServerInfo_INITIALIZING), stats)
	if err != nil {
		t.Errorf("regex match err %v", err)
	}
	state2Matched, err := regexp.MatchString(fmt.Sprintf("%s%d", prefix, envoy_admin_v2alpha.ServerInfo_DRAINING), stats2)
	if err != nil {
		t.Errorf("regex match err %v", err)
	}
	state3Matched, err := regexp.MatchString(fmt.Sprintf("%s%d", prefix, envoy_admin_v2alpha.ServerInfo_LIVE), stats3)
	if err != nil {
		t.Errorf("regex match err %v", err)
	}
	state4Matched, err := regexp.MatchString(fmt.Sprintf("%s%d", prefix, envoy_admin_v2alpha.ServerInfo_PRE_INITIALIZING), stats4)
	if err != nil {
		t.Errorf("regex match err %v", err)
	}

	if !(stateMatched &&
		state2Matched &&
		state3Matched &&
		state4Matched) {
		t.Error("mosn state is not expected", stateMatched, state2Matched, state3Matched, state4Matched)
	}

}

func TestDumpStatsForIstio(t *testing.T) {
	ads := &AdsConfig{
		converter: conv.NewConverter(),
	}
	getStatsForIstio := func() (statsInfo string, err error) {
		w := httptest.NewRecorder()
		ads.statsForIstio(w, nil)

		if w.Code != http.StatusOK {
			return "", errors.New("get mosn stats for istio failed")
		}
		b, err := ioutil.ReadAll(w.Body)
		if err != nil {
			return "", err
		}

		return string(b), nil
	}
	stats := ads.converter.Stats()
	stats.CdsUpdateSuccess.Inc(1)
	stats.CdsUpdateReject.Inc(2)
	stats.LdsUpdateSuccess.Inc(3)
	stats.LdsUpdateReject.Inc(4)

	statsForIstio, err := getStatsForIstio()
	if err != nil {
		t.Errorf("get stats for istio failed: %v", err)
	}
	match, _ := regexp.MatchString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_SUCCESS, 1), statsForIstio)
	match2, _ := regexp.MatchString(fmt.Sprintf("%s: %d\n", CDS_UPDATE_REJECT, 2), statsForIstio)
	match3, _ := regexp.MatchString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_SUCCESS, 3), statsForIstio)
	match4, _ := regexp.MatchString(fmt.Sprintf("%s: %d\n", LDS_UPDATE_REJECT, 4), statsForIstio)

	if !match ||
		!match2 ||
		!match3 ||
		!match4 {
		t.Error("wrong stats for istio output", match, match2, match3, match4)
	}

}
