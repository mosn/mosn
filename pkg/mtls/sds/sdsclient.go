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
	"errors"
	"sync"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type SdsClientImpl struct {
	SdsConfigMap   map[string]*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig
	SdsCallbackMap map[string]types.SdsUpdateCallbackFunc
	updatedLock    sync.Mutex
	sdsSubscriber  *SdsSubscriber
}

var sdsClient *SdsClientImpl
var sdsClientLock sync.Mutex
var sdsPostCallback func() = nil

var ErrSdsClientNotInit = errors.New("sds client not init")

// NewSdsClientSingleton use by tls module , when get sds config from xds
func NewSdsClientSingleton(config *envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig) types.SdsClient {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()
	if sdsClient != nil {
		// update sds config
		sdsClient.sdsSubscriber.sdsConfig = config.SdsConfig
		return sdsClient
	}

	sdsClient = &SdsClientImpl{
		SdsConfigMap:   make(map[string]*envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig),
		SdsCallbackMap: make(map[string]types.SdsUpdateCallbackFunc),
	}
	// For Istio , sds config should be the same
	// So we use first sds config to init sds subscriber
	sdsClient.sdsSubscriber = NewSdsSubscriber(sdsClient, config.SdsConfig, types.GetGlobalXdsInfo().ServiceNode, types.GetGlobalXdsInfo().ServiceCluster)
	utils.GoWithRecover(sdsClient.sdsSubscriber.Start, nil)
	return sdsClient
}

// CloseSdsClient used only mosn exit
func CloseSdsClient() {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()
	if sdsClient != nil && sdsClient.sdsSubscriber != nil {
		sdsClient.sdsSubscriber.Stop()
		sdsClient.sdsSubscriber = nil
		sdsClient = nil
	}
}

func (client *SdsClientImpl) AddUpdateCallback(sdsConfig *envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig, callback types.SdsUpdateCallbackFunc) error {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	client.SdsConfigMap[sdsConfig.Name] = sdsConfig
	client.SdsCallbackMap[sdsConfig.Name] = callback
	client.sdsSubscriber.SendSdsRequest(sdsConfig.Name)
	return nil
}

// DeleteUpdateCallback ...
func (client *SdsClientImpl) DeleteUpdateCallback(sdsConfig *envoy_extensions_transport_sockets_tls_v3.SdsSecretConfig) error {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	delete(client.SdsConfigMap, sdsConfig.Name)
	delete(client.SdsCallbackMap, sdsConfig.Name)
	return nil
}

// SetSecret invoked when sds subscriber get secret response
func (client *SdsClientImpl) SetSecret(name string, secret *envoy_extensions_transport_sockets_tls_v3.Secret) {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	if fc, ok := client.SdsCallbackMap[name]; ok {
		log.DefaultLogger.Debugf("[xds] [sds client],set secret = %v", name)
		mosnSecret := types.SecretConvert(secret)
		fc(name, mosnSecret)
	}
}

// SetSdsPostCallback ..
func SetSdsPostCallback(fc func()) {
	sdsPostCallback = fc
}

//////
///// Deprecated with xDS2
/////

type SdsClientImplDeprecated struct {
	SdsConfigMap   map[string]*envoy_api_v2_auth.SdsSecretConfig
	SdsCallbackMap map[string]types.SdsUpdateCallbackFunc
	updatedLock    sync.Mutex
	sdsSubscriber  *SdsSubscriberDeprecated
}

var sdsClientDeprecated *SdsClientImplDeprecated

// NewSdsClientSingletonDeprecated use by tls module , when get sds config from xds
func NewSdsClientSingletonDeprecated(config *envoy_api_v2_auth.SdsSecretConfig) types.SdsClientDeprecated {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()
	if sdsClientDeprecated != nil {
		// update sds config
		sdsClientDeprecated.sdsSubscriber.sdsConfig = config.SdsConfig
		return sdsClientDeprecated
	}

	sdsClientDeprecated = &SdsClientImplDeprecated{
		SdsConfigMap:   make(map[string]*envoy_api_v2_auth.SdsSecretConfig),
		SdsCallbackMap: make(map[string]types.SdsUpdateCallbackFunc),
	}
	// For Istio , sds config should be the same
	// So we use first sds config to init sds subscriber
	sdsClientDeprecated.sdsSubscriber = NewSdsSubscriberDeprecated(sdsClientDeprecated, config.SdsConfig, types.GetGlobalXdsInfo().ServiceNode, types.GetGlobalXdsInfo().ServiceCluster)
	utils.GoWithRecover(sdsClientDeprecated.sdsSubscriber.Start, nil)
	return sdsClientDeprecated
}

func (client *SdsClientImplDeprecated) AddUpdateCallbackDeprecated(sdsConfig *envoy_api_v2_auth.SdsSecretConfig, callback types.SdsUpdateCallbackFunc) error {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	client.SdsConfigMap[sdsConfig.Name] = sdsConfig
	client.SdsCallbackMap[sdsConfig.Name] = callback
	client.sdsSubscriber.SendSdsRequest(sdsConfig.Name)
	return nil
}

// DeleteUpdateCallbackDeprecated ...
func (client *SdsClientImplDeprecated) DeleteUpdateCallbackDeprecated(sdsConfig *envoy_api_v2_auth.SdsSecretConfig) error {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	delete(client.SdsConfigMap, sdsConfig.Name)
	delete(client.SdsCallbackMap, sdsConfig.Name)
	return nil
}

// SetSecretDeprecated invoked when sds subscriber get secret response
func (client *SdsClientImplDeprecated) SetSecretDeprecated(name string, secret *envoy_api_v2_auth.Secret) {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	if fc, ok := client.SdsCallbackMap[name]; ok {
		log.DefaultLogger.Debugf("[xds] [sds client],set secret = %v", name)
		mosnSecret := types.SecretConvertDeprecated(secret)
		fc(name, mosnSecret)
	}
}
