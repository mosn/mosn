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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/mtls/sds"
	"mosn.io/mosn/pkg/types"
)

func InitSdsSecertConfig(sdsUdsPath string) proto.Message {
	gRPCConfig := &envoy_config_core_v3.GrpcService_GoogleGrpc{
		TargetUri:  sdsUdsPath,
		StatPrefix: "sds-prefix",
		ChannelCredentials: &envoy_config_core_v3.GrpcService_GoogleGrpc_ChannelCredentials{
			CredentialSpecifier: &envoy_config_core_v3.GrpcService_GoogleGrpc_ChannelCredentials_LocalCredentials{
				LocalCredentials: &envoy_config_core_v3.GrpcService_GoogleGrpc_GoogleLocalCredentials{},
			},
		},
		CallCredentials: []*envoy_config_core_v3.GrpcService_GoogleGrpc_CallCredentials{
			{
				CredentialSpecifier: &envoy_config_core_v3.GrpcService_GoogleGrpc_CallCredentials_GoogleComputeEngine{},
			},
		},
	}
	source := &envoy_config_core_v3.ConfigSource{
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
	}
	return source
}

func Test_SdsClient(t *testing.T) {
	// init prepare
	sdsUdsPath := "/tmp/sds2"
	sds.SubscriberRetryPeriod = 500 * time.Millisecond
	defer func() {
		sds.SubscriberRetryPeriod = 3 * time.Second
	}()
	callback := 0
	sds.SetSdsPostCallback(func() {
		callback++
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

	t.Run("test add update callback", func(t *testing.T) {
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
	})

	t.Run("test fetch secrets", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		secret, err := sdsClient.FetchSecret(ctx, "default")
		require.Nil(t, err)
		require.Equal(t, "default", secret.Name)
		time.Sleep(time.Second)
		// FetchSecret should not take effects on callback
		require.Equal(t, 1, callback)
	})

	t.Run("test require secret", func(t *testing.T) {
		sdsClient.RequireSecret("default")
		select {
		case <-updatedChan:
			if callback != 2 {
				t.Fatalf("sds post callback unexpected")
			}
		case <-time.After(time.Second * 10):
			t.Errorf("callback reponse timeout")
		}
	})
}

func TestSdsWithDifferentConfig(t *testing.T) {
	// init prepare
	sdsUdsPath := "/tmp/sds3"
	sds.SubscriberRetryPeriod = 500 * time.Millisecond
	defer func() {
		sds.SubscriberRetryPeriod = 3 * time.Second
	}()
	// mock sds server
	srv := InitMockSdsServer(sdsUdsPath, t)
	defer srv.Stop()
	config := InitSdsSecertConfig(sdsUdsPath)
	defer sds.CloseSdsClient()
	// wait server start
	time.Sleep(time.Second)
	// register callback
	finish := make(chan struct{}, 1)
	mtls.RegisterSdsCallback("finish", func(c *v2.TLSConfig) {
		select {
		case finish <- struct{}{}:
			log.DefaultLogger.Infof("receive a callback, send finish: %+v", c)
		default:
			log.DefaultLogger.Errorf("receive a callback, but channel is full, %+v", c)
		}
	})
	// listener from sds
	sdsConfig := &v2.SdsConfig{
		CertificateConfig: &v2.SecretConfigWrapper{
			Name:      "default",
			SdsConfig: config,
		},
		ValidationConfig: &v2.SecretConfigWrapper{
			Name:      "rootca",
			SdsConfig: config,
		},
	}

	serverConfig := v2.TLSConfig{
		Status:    true,
		SdsConfig: sdsConfig,
		Callbacks: []string{"finish"}, // Add a callback for certificate update success
	}
	lc := &v2.Listener{}
	lc.Name = "server_sds"
	lc.FilterChains = []v2.FilterChain{
		{
			TLSContexts: []v2.TLSConfig{
				serverConfig,
			},
		},
	}
	ctxMng, err := mtls.NewTLSServerContextManager(lc)
	require.Nil(t, err)
	tls_server := mtls.MockServer{
		Mng: ctxMng,
	}
	tls_server.GoListenAndServe()
	defer tls_server.Close()
	time.Sleep(time.Second) //wait server start
	// make sure certificate is received
	select {
	case <-finish:
	case <-time.After(5 * time.Second):
		t.Fatal("no certificate callback received")
	}
	// a same sds config's certificate with different tls config
	t.Run("call server, verify success", func(t *testing.T) {
		cltMng, err := mtls.NewTLSClientContextManager("verify_succes", &v2.TLSConfig{
			Status:     true,
			SdsConfig:  sdsConfig,
			ServerName: "127.0.0.1",
		})
		require.Nil(t, err)
		resp, err := mtls.MockClient(tls_server.Addr, cltMng)
		require.Nil(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		// release sources
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	})

	t.Run("call server, verify failed", func(t *testing.T) {
		cltMng, err := mtls.NewTLSClientContextManager("verify_failed", &v2.TLSConfig{
			Status:     true,
			SdsConfig:  sdsConfig,
			ServerName: "192.168.1.1",
		})
		require.Nil(t, err)
		_, err = mtls.MockClient(tls_server.Addr, cltMng)
		require.NotNil(t, err)
	})

}

const sdsJson = `{
	"name":"default",
	"sdsConfig":{
		"apiConfigSource":{
			"apiType":"GRPC",
			"grpcServices":[
				{
					"googleGrpc": {
						"callCredentials": [{"googleComputeEngine": {}}],
						"channelCredentials": {"localCredentials":{}},
						"statPrefix":"sds-prefix",
						"targetUri":"@/var/run/test"
					},

				}
			]
		}
	}
}`

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
	t.Run("got from xds", func(t *testing.T) {
		scw := &v2.SecretConfigWrapper{
			Name:      "default",
			SdsConfig: InitSdsSecertConfig("@/var/run/test"),
		}
		sdsConfig, err := ConvertConfig(scw.SdsConfig)
		require.Nil(t, err)
		require.Equal(t, "@/var/run/test", sdsConfig.sdsUdsPath)
		require.Equal(t, "sds-prefix", sdsConfig.statPrefix)

		// To Json string
		b, err := json.Marshal(scw)
		require.Nil(t, err)
		fmt.Println(string(b))
	})
}
