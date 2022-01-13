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
	v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"sync"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type SdsClientImpl struct {
	SdsCallbackMap map[string]types.SdsUpdateCallbackFunc
	updatedLock    sync.Mutex
	sdsSubscriber  *SdsSubscriber
	sdsClient      v2.SecretDiscoveryServiceClient
	sdsConfig      interface{}
}

var sdsClient *SdsClientImpl
var sdsClientLock sync.Mutex
var sdsPostCallback func() = nil

var ErrSdsClientNotInit = errors.New("sds client not init")

// TODO: support sds client index instead of singleton
func NewSdsClientSingleton(cfg interface{}) types.SdsClient {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()

	if sdsClient != nil {
		// update sds config
		sdsClient.sdsSubscriber.sdsConfig = cfg
		sdsClient.sdsConfig = cfg
		return sdsClient
	} else {
		sdsClient = &SdsClientImpl{
			SdsCallbackMap: make(map[string]types.SdsUpdateCallbackFunc),
			sdsConfig:      cfg,
		}
		// For Istio , sds config should be the same
		// So we use first sds config to init sds subscriber
		sdsClient.sdsSubscriber = NewSdsSubscriber(sdsClient, cfg)
		utils.GoWithRecover(sdsClient.sdsSubscriber.Start, nil)
		return sdsClient
	}
}

func CloseSdsClient() {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()
	if sdsClient != nil && sdsClient.sdsSubscriber != nil {
		log.DefaultLogger.Warnf("[mtls] sds client stopped")
		sdsClient.sdsSubscriber.Stop()
		sdsClient.sdsSubscriber = nil
		sdsClient = nil
	}
}

func (client *SdsClientImpl) AddUpdateCallback(name string, callback types.SdsUpdateCallbackFunc) error {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	client.SdsCallbackMap[name] = callback
	client.sdsSubscriber.SendSdsRequest(name)
	return nil
}

// DeleteUpdateCallback
func (client *SdsClientImpl) DeleteUpdateCallback(name string) error {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	delete(client.SdsCallbackMap, name)
	return nil
}

// SetSecret invoked when sds subscriber get secret response
func (client *SdsClientImpl) SetSecret(name string, secret *types.SdsSecret) {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	if fc, ok := client.SdsCallbackMap[name]; ok {
		log.DefaultLogger.Debugf("[xds] [sds client],set secret = %v", name)
		fc(name, secret)
	}
}

// SetPostCallback
func SetSdsPostCallback(fc func()) {
	sdsPostCallback = fc
}

func (client *SdsClientImpl) GetSdsClient() v2.SecretDiscoveryServiceClient {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	if client.sdsClient != nil {
		return client.sdsClient
	}
	nativeClient, err := GetSdsNativeClient(client.sdsConfig)
	if err == nil {
		log.DefaultLogger.Errorf("[xds] [sds client],native client init failed,error = %v", err)
		return nil
	}
	return nativeClient
}

func (client *SdsClientImpl) GetSdsSubscriber() *SdsSubscriber {
	return client.sdsSubscriber
}
