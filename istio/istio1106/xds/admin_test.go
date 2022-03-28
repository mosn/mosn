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

	envoyControlPlaneAPI "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/mosn/istio/istio1106/xds/conv"
	"mosn.io/mosn/pkg/stagemanager"
)

func TestGetState(t *testing.T) {
	ads := &AdsConfig{
		converter: conv.NewConverter(),
	}
	getMosnStateForIstio := func() (state envoyControlPlaneAPI.ServerInfo_State, err error) {
		w := httptest.NewRecorder()
		serverInfoForIstio(w, nil)

		if w.Code != http.StatusOK {
			return 0, errors.New("get mosn states for istio failed")
		}
		serverInfo := envoyControlPlaneAPI.ServerInfo{}
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

	verifyMosnState2IstioState := func(state stagemanager.State, expectedState envoyControlPlaneAPI.ServerInfo_State) {
		stateForIstio, err := getMosnStateForIstio()
		if err != nil {
			t.Fatalf("get mosn states for istio failed: %v", err)
		}
		stats, err := getStatsForIstio()
		if err != nil {
			t.Fatal("get mosn stats for istio failed")
		}
		if stateForIstio != expectedState {
			t.Errorf("unexpected istio state %v while expected %v from state %v", stateForIstio, expectedState, state)
		}

		prefix := fmt.Sprintf("%s: ", SERVER_STATE)
		stateMatched, err := regexp.MatchString(fmt.Sprintf("%s%d", prefix, expectedState), stats)
		if err != nil {
			t.Errorf("regex match err %v", err)
		}
		if !stateMatched {
			t.Errorf("not found state %v from stats: %v", expectedState, stats)
		}
	}

	// verify init state
	verifyMosnState2IstioState(stagemanager.Nil, envoyControlPlaneAPI.ServerInfo_PRE_INITIALIZING)
	// verify set state
	stm := stagemanager.InitStageManager(nil, "", nil)
	for mosnState, istioState := range mosnState2IstioState {
		stm.SetState(mosnState)
		verifyMosnState2IstioState(mosnState, istioState)
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

func TestEnvoyConfigDump(t *testing.T) {
	// TODO: add it
}
