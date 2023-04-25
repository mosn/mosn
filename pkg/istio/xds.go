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

package istio

import (
	"math/rand"
	"sync"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/utils"
)

type XdsStreamConfig interface {
	CreateXdsStreamClient() (XdsStreamClient, error)
	RefreshDelay() time.Duration
	// InitAdsRequest returns the first request for the ads
	InitAdsRequest() interface{}
}

type ADSClient struct {
	streamClientMutex sync.RWMutex
	streamClient      XdsStreamClient
	config            XdsStreamConfig
	stopChan          chan struct{}
	sendTimer         *time.Timer
}

func NewAdsClient(config *v2.MOSNConfig) (*ADSClient, error) {
	cfg, err := ParseAdsConfig(config.RawDynamicResources, config.RawStaticResources)
	if err != nil {
		return nil, err
	}
	return &ADSClient{
		config:    cfg,
		stopChan:  make(chan struct{}),
		sendTimer: time.NewTimer(0),
	}, nil
}

type XdsStreamClient interface {
	Send(req interface{}) error
	Recv() (interface{}, error)
	HandleResponse(resp interface{})
	Stop()
}

func (adsClient *ADSClient) GetStreamClient() (c XdsStreamClient) {
	adsClient.streamClientMutex.RLock()
	c = adsClient.streamClient
	adsClient.streamClientMutex.RUnlock()
	return
}

func (adsClient *ADSClient) Start() {
	if adsClient.config == nil {
		log.StartLogger.Infof("[xds] no xds config parsed, no xds action")
		return
	}
	_ = adsClient.connect()
	utils.GoWithRecover(adsClient.sendRequestLoop, nil)
	utils.GoWithRecover(adsClient.receiveResponseLoop, nil)
}

func (adsClient *ADSClient) sendRequestLoop() {
	log.DefaultLogger.Debugf("[xds] [ads client] send request, start with cds")
	for {
		select {
		case <-adsClient.stopChan:
			log.DefaultLogger.Infof("[xds] [ads client] send request loop shutdown")
			adsClient.stopStreamClient()
			return
		case <-adsClient.sendTimer.C:
			c := adsClient.GetStreamClient()
			if c == nil {
				log.DefaultLogger.Infof("[xds] [ads client] stream client closed, sleep 1s and wait for reconnect")
				time.Sleep(time.Second)
				adsClient.reconnect()
			} else if err := c.Send(adsClient.config.InitAdsRequest()); err != nil {
				log.DefaultLogger.Infof("[xds] [ads client] send thread request cds fail!auto retry next period")
				adsClient.reconnect()
			}
			adsClient.sendTimer.Reset(adsClient.config.RefreshDelay())
		}
	}
}

func (adsClient *ADSClient) receiveResponseLoop() {
	for {
		select {
		case <-adsClient.stopChan:
			log.DefaultLogger.Infof("[xds] [ads client] receive response loop shutdown")
			return
		default:
			c := adsClient.GetStreamClient()
			if c == nil {
				log.DefaultLogger.Infof("[xds] [ads client] try receive response: stream client closed")
				time.Sleep(time.Second)
				continue
			}
			resp, err := c.Recv()
			if err != nil {
				log.DefaultLogger.Infof("[xds] [ads client] get resp error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			c.HandleResponse(resp)
		}
	}
}

var disableReconnect bool

func DisableReconnect() {
	disableReconnect = true
}

func EnableReconnect() {
	disableReconnect = false
}

const maxRertyInterval = 60 * time.Second

func computeInterval(t time.Duration) time.Duration {
	t = t * 2
	if t >= maxRertyInterval {
		t = maxRertyInterval
	}
	return t
}

func (adsClient *ADSClient) reconnect() {
	adsClient.stopStreamClient()
	log.DefaultLogger.Infof("[xds] [ads client] close stream client before retry")
	interval := time.Second

	for {
		if !disableReconnect {
			err := adsClient.connect()
			if err == nil {
				log.DefaultLogger.Infof("[xds] [ads client] stream client reconnected")
				return
			}
			log.DefaultLogger.Infof("[xds] [ads client] stream client reconnect failed %v,  retry after %v", err, interval)
		}
		// sleep random
		time.Sleep(interval + time.Duration(rand.Intn(1000))*time.Millisecond)
		interval = computeInterval(interval)
	}
}

func (adsClient *ADSClient) stopStreamClient() {
	adsClient.streamClientMutex.Lock()
	if adsClient.streamClient != nil {
		adsClient.streamClient.Stop()
		adsClient.streamClient = nil
	}
	adsClient.streamClientMutex.Unlock()
}

func (adsClient *ADSClient) connect() error {
	adsClient.streamClientMutex.Lock()
	defer adsClient.streamClientMutex.Unlock()

	if adsClient.streamClient != nil {
		return nil
	}
	client, err := adsClient.config.CreateXdsStreamClient()
	if err != nil {
		return err
	}
	adsClient.streamClient = client

	return nil
}

func (adsClient *ADSClient) Stop() {
	close(adsClient.stopChan)
}

// close stream client and trigger reconnect right now
func (adsClient *ADSClient) ReconnectStreamClient() {
	adsClient.stopStreamClient()
	log.DefaultLogger.Infof("[xds] [ads client] close stream client")

	if disableReconnect {
		log.DefaultLogger.Infof("[xds] [ads client] stream client reconnect disabled")
		return
	}

	adsClient.sendTimer.Reset(0)
}
