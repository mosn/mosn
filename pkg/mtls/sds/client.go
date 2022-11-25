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
	"errors"
	"sync"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type SdsClientImpl struct {
	SdsCallbackMap map[string]types.SdsUpdateCallbackFunc
	updatedLock    sync.Mutex
	sdsSubscriber  *SdsSubscriber
}

var sdsClient *SdsClientImpl
var sdsClientLock sync.Mutex

// sdsPostCallback usually used in test scenarios to sense that the asynchronous certificate response has completed the push
var sdsPostCallback func() = nil

var ErrSdsClientNotInit = errors.New("sds client not init")

// TODO: support sds client index instead of singleton
func NewSdsClientSingleton(cfg interface{}) types.SdsClient {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()

	if sdsClient != nil {
		// update sds config
		sdsClient.sdsSubscriber.sdsConfig = cfg
		return sdsClient
	} else {
		sdsClient = &SdsClientImpl{
			SdsCallbackMap: make(map[string]types.SdsUpdateCallbackFunc),
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

func (client *SdsClientImpl) RequireSecret(name string) {
	client.sdsSubscriber.SendSdsRequest(name)
}

func (client *SdsClientImpl) FetchSecret(ctx context.Context, name string) (*types.SdsSecret, error) {
	return client.sdsSubscriber.FetchSdsSecret(ctx, name)
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

// AckResponse invoked when sds subscriber receive a response
func (client *SdsClientImpl) AckResponse(resp interface{}) {
	client.sdsSubscriber.SendAck(resp)
}

// SetPostCallback
func SetSdsPostCallback(fc func()) {
	sdsPostCallback = fc
}
