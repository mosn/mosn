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

package stagemanager_test

import (
	"encoding/json"
	"testing"

	"github.com/urfave/cli"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/mosn"
	"mosn.io/mosn/pkg/stagemanager"
)

// TODO: may be use a common test config
// test new config
const mosnConfig = `{
	"servers":[{
		"default_log_path": "stdout",
		"listeners":[{
			"name":"serverListener",
			"address": "127.0.0.1:8080",
			"bind_port": true,
			"filter_chains": [{
				"filters": [
					{
						"type": "proxy",
						"config": {
							"downstream_protocol": "Http1",
							"upstream_protocol": "Http1",
							"router_config_name":"server_router"
						}
					},
					{
						"type": "connection_manager",
						"config":{
							"router_config_name":"server_router",
							"virtual_hosts":[{
								"domains": ["*"],
								"routers": [{
									"match":{"prefix":"/"},
									"route":{"cluster_name":"serverCluster"}
								}]
							}]
						}
					}
				]
			}]
		}],
		"routers": [
			{
				"router_config_name":"server_router",
				"virtual_hosts":[
					{
						"name": "virtualhost00",
						"domains": ["www.server.com"],
						"routers": [{
							"match":{"prefix":"/"},
						 	"route":{"cluster_name":"serverCluster"}
						}]
					},
					{
						"name": "virtualhost01",
						 "domains": ["www.test.com"],
						 "routers": [{
							 "match":{"prefix":"/"},
							 "route":{"cluster_name":"serverCluster"}
						 }]
					}
				]
			},
			{
				 "router_config_name":"test_router",
				 "virtual_hosts":[{
					 "domains": ["*"],
					 "routers": [{
						 "match":{"prefix":"/"},
						 "route":{"cluster_name":"serverCluster"}
					 }]
				 }]
			}
		]
	}],
	"cluster_manager":{
		"clusters":[{
			"name":"serverCluster",
			"type": "SIMPLE",
			"lb_type": "LB_RANDOM",
			"hosts":[]
		}]
	}
}`

func TestStageManager(t *testing.T) {
	stm := stagemanager.InitStageManager(&cli.Context{}, "", mosn.NewMosn())
	// test for mock
	testCall := 0
	configmanager.RegisterConfigLoadFunc(func(p string) *v2.MOSNConfig {
		if testCall != 2 {
			t.Errorf("load config is not called after 2 stages registered")
		}
		testCall = 0

		cfg := &v2.MOSNConfig{}
		content := []byte(mosnConfig)
		if err := json.Unmarshal(content, cfg); err != nil {
			t.Fatal(err)
		}
		return cfg
	})
	defer configmanager.RegisterConfigLoadFunc(configmanager.DefaultConfigLoad)
	stm.AppendPreStartStage(mosn.DefaultPreStartStage)
	stm.AppendParamsParsedStage(func(_ *cli.Context) {
		testCall++
		if testCall != 1 {
			t.Errorf("unexpected params parsed stage call: 1")
		}
	}).AppendParamsParsedStage(func(_ *cli.Context) {
		testCall++
		if testCall != 2 {
			t.Errorf("unexpected params parsed stage call: 2")
		}
	}).AppendInitStage(func(_ *v2.MOSNConfig) {
		// testCall reset to 0 in the new registered ConfigLoad
		testCall++
		if testCall != 1 {
			t.Errorf("init stage call, expect 1 while got %v", testCall)
		}
	}).AppendPreStartStage(func(_ stagemanager.Mosn) {
		testCall++
		if testCall != 2 {
			t.Errorf("pre start stage call, expect 2 while got %v", testCall)
		}
	}).AppendStartStage(func(_ stagemanager.Mosn) {
		testCall++
		if testCall != 3 {
			t.Errorf("start stage call, expect 3 while got %v", testCall)
		}
	}).AppendAfterStartStage(func(_ stagemanager.Mosn) {
		testCall++
		if testCall != 4 {
			t.Errorf("after start stage call, expect 4 while got %v", testCall)
		}
	}).AppendPreStopStage(func(_ stagemanager.Mosn) {
		testCall++
		if testCall != 5 {
			t.Errorf("pre stop stage call, expect 5 while got %v", testCall)
		}
	}).AppendAfterStopStage(func(_ stagemanager.Mosn) {
		testCall++
		if testCall != 6 {
			t.Errorf("after stop stage call, expect 6 while got %v", testCall)
		}
	})
	if testCall != 0 {
		t.Errorf("should call nothing")
	}
	stm.Run()
	if !(testCall == 4 &&
		stagemanager.GetState() == stagemanager.AfterStart) {
		t.Errorf("run stage failed, testCall: %v, stage: %v", testCall, stagemanager.GetState())
	}
	stagemanager.Notice(stagemanager.GracefulQuit)
	stm.WaitFinish()
	if !(testCall == 4 &&
		stagemanager.GetState() == stagemanager.Running) {
		t.Errorf("wait stage failed, testCall: %v, stage: %v", testCall, stagemanager.GetState())
	}
	stm.Stop()
	if !(testCall == 6 &&
		stagemanager.GetState() == stagemanager.Stopped) {
		t.Errorf("stop stage failed, testCall: %v, stage: %v", testCall, stagemanager.GetState())
	}
}
