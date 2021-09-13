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
	"bytes"
	"encoding/json"
	"testing"
	"time"

	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/gogo/protobuf/jsonpb"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls/sds"
	"mosn.io/mosn/pkg/types"
)

func InitSdsSecertConfig(sdsUdsPath string) interface{} {
	gRPCConfig := &envoy_api_v2_core.GrpcService_GoogleGrpc{
		TargetUri:  sdsUdsPath,
		StatPrefix: "sds-prefix",
		ChannelCredentials: &envoy_api_v2_core.GrpcService_GoogleGrpc_ChannelCredentials{
			CredentialSpecifier: &envoy_api_v2_core.GrpcService_GoogleGrpc_ChannelCredentials_LocalCredentials{
				LocalCredentials: &envoy_api_v2_core.GrpcService_GoogleGrpc_GoogleLocalCredentials{},
			},
		},
	}
	source := &envoy_api_v2_core.ConfigSource{
		ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &envoy_api_v2_core.ApiConfigSource{
				ApiType: envoy_api_v2_core.ApiConfigSource_GRPC,
				GrpcServices: []*envoy_api_v2_core.GrpcService{
					{
						TargetSpecifier: &envoy_api_v2_core.GrpcService_GoogleGrpc_{
							GoogleGrpc: gRPCConfig,
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	marshaler := &jsonpb.Marshaler{}
	_ = marshaler.Marshal(&buf, source)
	m := map[string]interface{}{}
	json.Unmarshal(buf.Bytes(), &m)
	return m

}

func Test_AddUpdateCallback(t *testing.T) {
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
	srv := InitMockSdsServer(sdsUdsPath, t)
	defer srv.Stop()
	config := InitSdsSecertConfig(sdsUdsPath)
	sdsClient := sds.NewSdsClientSingleton(config)
	defer sds.CloseSdsClient()

	// wait server start and stop makes reconnect
	time.Sleep(time.Second)
	srv.Stop()
	// wait server connection stop
	time.Sleep(time.Second)
	// send request
	updatedChan := make(chan int, 1) // do not block the update channel
	log.DefaultLogger.Infof("add update callback")

	sdsClient.AddUpdateCallback("default", func(name string, secret *types.SdsSecret) {
		if name != "default" {
			t.Errorf("name should same with config.name")
		}
		log.DefaultLogger.Infof("update callback is called")
		updatedChan <- 1
	})
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
