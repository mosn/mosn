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
	"fmt"
	"sync"
	"time"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

type SdsSubscriber struct {
	provider             types.SecretProvider
	reqQueue             chan string
	ackChan              chan interface{}
	sdsConfig            interface{}
	sdsStreamClient      SdsStreamClient
	sdsStreamClientMutex sync.RWMutex
	stopChannel          chan struct{}
}

type SdsStreamClient interface {
	// Send creates a secret discovery request with name, and send it to server
	Send(name string) error
	// Recv receives a secret discovery response, handle it and send an ack response
	Recv(provider types.SecretProvider, callback func()) error
	// Fetch creates a secret discovery request with name, and wait the response
	Fetch(ctx context.Context, name string) (*types.SdsSecret, error)
	// AckResponse creates an ack request based on the response
	AckResponse(resp interface{})
	// Stop stops a stream client
	Stop()
}

func NewSdsSubscriber(provider types.SecretProvider, sdsConfig interface{}) *SdsSubscriber {
	return &SdsSubscriber{
		provider:    provider,
		sdsConfig:   sdsConfig,
		reqQueue:    make(chan string, 10),
		ackChan:     make(chan interface{}, 10),
		stopChannel: make(chan struct{}),
	}
}

var SubscriberRetryPeriod = 3 * time.Second

func (subscribe *SdsSubscriber) Start() {
	subscribe.connect()

	utils.GoWithRecover(func() {
		subscribe.sendRequestLoop()
	}, nil)
	utils.GoWithRecover(func() {
		subscribe.receiveResponseLoop()
	}, nil)

}

func (subscribe *SdsSubscriber) getSdsStreamClient(config interface{}) error {
	subscribe.sdsStreamClientMutex.Lock()
	defer subscribe.sdsStreamClientMutex.Unlock()
	if subscribe.sdsStreamClient != nil {
		return nil
	}
	streamClient, err := GetSdsStreamClient(subscribe.sdsConfig)
	if err != nil {
		return err
	}
	subscribe.sdsStreamClient = streamClient
	return nil
}

func (subscribe *SdsSubscriber) connect() {
	for {
		if err := subscribe.getSdsStreamClient(subscribe.sdsConfig); err != nil {
			time.Sleep(SubscriberRetryPeriod)
			continue
		}
		log.DefaultLogger.Infof("[sds][subscribe] init sds stream client success")
		return
	}
}

func (subscribe *SdsSubscriber) Stop() {
	close(subscribe.stopChannel)
}

func (subscribe *SdsSubscriber) SendSdsRequest(name string) {
	subscribe.reqQueue <- name
}

func (subscribe *SdsSubscriber) FetchSdsSecret(ctx context.Context, name string) (*types.SdsSecret, error) {
	subscribe.sdsStreamClientMutex.RLock()
	clt := subscribe.sdsStreamClient
	subscribe.sdsStreamClientMutex.RUnlock()
	if clt == nil {
		return nil, fmt.Errorf("fetch secret %s failed, because the sds stream client is not ready yet", name)
	}
	return clt.Fetch(ctx, name)
}

func (subscribe *SdsSubscriber) SendAck(resp interface{}) {
	subscribe.ackChan <- resp
}

var retryInterval = time.Second

func (subscribe *SdsSubscriber) sendRequestLoop() {
	for {
		select {
		case <-subscribe.stopChannel:
			log.DefaultLogger.Errorf("[xds] [sds subscriber] send request loop closed")
			subscribe.cleanSdsStreamClient()
			return
		case name := <-subscribe.reqQueue:
			for {
				subscribe.sdsStreamClientMutex.RLock()
				clt := subscribe.sdsStreamClient
				subscribe.sdsStreamClientMutex.RUnlock()

				if clt == nil {
					log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed, send request failed")
					time.Sleep(retryInterval)
					continue
				}

				if err := clt.Send(name); err != nil {
					log.DefaultLogger.Alertf("sds.subscribe.request", "[xds] [sds subscriber] send sds request fail , resource name = %v", name)
					time.Sleep(retryInterval)
					continue
				}
				break
			}
		case ack := <-subscribe.ackChan:
			// send ack should always on the original connection(sdsStreamClient)
			// so if get sdsStreamClient failed, we will ignore the ack
			subscribe.sdsStreamClientMutex.RLock()
			clt := subscribe.sdsStreamClient
			subscribe.sdsStreamClientMutex.RUnlock()
			if clt != nil {
				clt.AckResponse(ack)
			}
		}
	}
}

func (subscribe *SdsSubscriber) receiveResponseLoop() {
	for {
		select {
		case <-subscribe.stopChannel:
			log.DefaultLogger.Errorf("[xds] [sds subscriber]  receive response loop closed")
			return
		default:
			subscribe.sdsStreamClientMutex.RLock()
			clt := subscribe.sdsStreamClient
			subscribe.sdsStreamClientMutex.RUnlock()

			if clt == nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed, reconnect after 1s")
				time.Sleep(retryInterval)
				subscribe.reconnect()
				continue
			}

			if err := clt.Recv(subscribe.provider, sdsPostCallback); err != nil {
				log.DefaultLogger.Infof("[xds] [sds subscriber] get resp error: %v,reconnect  1s", err)
				time.Sleep(retryInterval)
				subscribe.reconnect()
				continue
			}

		}
	}
}
func (subscribe *SdsSubscriber) reconnect() {
	subscribe.cleanSdsStreamClient()
	subscribe.connect()
}

func (subscribe *SdsSubscriber) cleanSdsStreamClient() {
	subscribe.sdsStreamClientMutex.Lock()
	defer subscribe.sdsStreamClientMutex.Unlock()
	if subscribe.sdsStreamClient != nil {
		subscribe.sdsStreamClient.Stop()
		subscribe.sdsStreamClient = nil
	}
	log.DefaultLogger.Infof("[xds] [sds subscriber] stream client closed")
}
