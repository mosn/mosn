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

package sds

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/sds"
	"mosn.io/mosn/pkg/types"
)

func InitSdsSecertConfigV3(sdsUdsPath string) *envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig {
	gRPCConfig := &envoy_config_core_v3.GrpcService_GoogleGrpc{
		TargetUri:  sdsUdsPath,
		StatPrefix: "sds-prefix",
		ChannelCredentials: &envoy_config_core_v3.GrpcService_GoogleGrpc_ChannelCredentials{
			CredentialSpecifier: &envoy_config_core_v3.GrpcService_GoogleGrpc_ChannelCredentials_LocalCredentials{
				LocalCredentials: &envoy_config_core_v3.GrpcService_GoogleGrpc_GoogleLocalCredentials{},
			},
		},
	}
	config := &envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig{
		Name: "default",
		SdsConfig: &envoy_config_core_v3.ConfigSource{
			ConfigSourceSpecifier: &envoy_config_core_v3.ConfigSource_ApiConfigSource{
				ApiConfigSource: &envoy_config_core_v3.ApiConfigSource{
					ApiType: envoy_config_core_v3.ApiConfigSource_GRPC,
					GrpcServices: []*envoy_config_core_v3.GrpcService{
						{
							TargetSpecifier: &envoy_config_core_v3.GrpcService_GoogleGrpc_{
								GoogleGrpc: gRPCConfig,
							},
						},
					},
				},
			},
		},
	}
	return config
}

func Test_AddUpdateCallbackV3(t *testing.T) {
	// init prepare
	sdsUdsPath := "/tmp/sds2"
	sds.SubscriberRetryPeriod = 500 * time.Millisecond
	defer func() {
		sds.SubscriberRetryPeriod = 3 * time.Second
	}()
	callback := 0
	sds.SetSdsPostCallback(func() {
		callback = 1
	})
	// mock sds server
	srv := InitMockSdsServerV3(sdsUdsPath, t)
	defer srv.Stop()
	config := InitSdsSecertConfigV3(sdsUdsPath)
	sdsClient := sds.NewSdsClientSingleton(config)
	if sdsClient == nil {
		t.Errorf("get sds client fail")
	}
	defer sdsClient.CloseSdsClient()

	// wait server start and stop makes reconnect
	time.Sleep(time.Second)
	srv.Stop()
	// wait server connection stop
	time.Sleep(time.Second)
	// send request
	updatedChan := make(chan int, 1) // do not block the update channel
	log.DefaultLogger.Infof(" add update callback")
	sdsClient.AddUpdateCallback(config, func(name string, secret *types.SdsSecret) {
		if name != "default" {
			t.Errorf("name should same with config.name")
		}
		log.DefaultLogger.Infof("update callback is called")
		updatedChan <- 1
	})
	// sdsClient.SetSecret(config.Name, &envoy_extensions_transport_sockets_tls_v3.Secret{})
	time.Sleep(time.Second)
	go func() {
		err := srv.Start()
		if !srv.started {
			t.Fatalf("%s start error: %v", sdsUdsPath, err)
		}
	}()
	select {
	case <-updatedChan:
		if callback != 1 {
			t.Fatalf("sds post callback unexpected")
		}
	case <-time.After(time.Second * 10):
		t.Errorf("callback reponse timeout")
	}
}

func equalJsonStr(s1, s2 string) bool {
	var o1 interface{}
	if err := json.Unmarshal([]byte(s1), &o1); err != nil {
		return false
	}
	var o2 interface{}
	if err := json.Unmarshal([]byte(s2), &o2); err != nil {
		return false
	}
	return reflect.DeepEqual(o1, o2)
}

func TestConvertFromJson(t *testing.T) {
	// TODO: add it , need a v3 sds config
}
