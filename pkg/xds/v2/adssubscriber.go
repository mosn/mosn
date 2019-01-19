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

package v2

import (
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
)

// Start adsClient send goroutine and receive goroutine
// send goroutine periodic request lds and cds
// receive goroutine handle response for both client request and server push
func (adsClient *ADSClient) Start() {
	adsClient.StreamClient = adsClient.AdsConfig.GetStreamClient()
	go adsClient.sendThread()
	go adsClient.receiveThread()
}

func (adsClient *ADSClient) sendThread() {
	log.DefaultLogger.Tracef("send thread request cds")
	err := adsClient.V2Client.reqClusters(adsClient.StreamClient)
	if err != nil {
		log.DefaultLogger.Warnf("send thread request cds fail!auto retry next period")
		adsClient.reconnect()
	}

	refreshDelay := adsClient.AdsConfig.RefreshDelay
	t1 := time.NewTimer(*refreshDelay)
	for {
		select {
		case <-adsClient.SendControlChan:
			log.DefaultLogger.Tracef("send thread receive graceful shut down signal")
			adsClient.AdsConfig.closeADSStreamClient()
			adsClient.StopChan <- 1
			return
		case <-t1.C:
			err := adsClient.V2Client.reqClusters(adsClient.StreamClient)
			if err != nil {
				log.DefaultLogger.Warnf("send thread request cds fail!auto retry next period")
				adsClient.reconnect()
			}
			t1.Reset(*refreshDelay)
		}
	}
}

func (adsClient *ADSClient) receiveThread() {
	for {
		select {
		case <-adsClient.RecvControlChan:
			log.DefaultLogger.Tracef("receive thread receive graceful shut down signal")
			adsClient.StopChan <- 2
			return
		default:
			if adsClient.StreamClient == nil {
				log.DefaultLogger.Warnf("stream client closed, sleep 1s and wait for reconnect")
				time.Sleep(time.Second)
				continue
			}
			resp, err := adsClient.StreamClient.Recv()
			if err != nil {
				log.DefaultLogger.Warnf("get resp timeout: %v, retry after 1s", err)
				time.Sleep(time.Second)
				continue
			}
			typeURL := resp.TypeUrl
			HandleTypeURL(typeURL, adsClient, resp)
		}
	}
}

func (adsClient *ADSClient) reconnect() {

	adsClient.AdsConfig.closeADSStreamClient()
	adsClient.StreamClient = nil
	log.DefaultLogger.Infof("stream client closed")

	for {
		adsClient.StreamClient = adsClient.AdsConfig.GetStreamClient()
		if adsClient.StreamClient == nil {
			log.DefaultLogger.Warnf("stream client reconnect failed, retry after 1s")
			time.Sleep(time.Second)
			continue
		}
		log.DefaultLogger.Infof("stream client reconnected")
		break
	}
}

// Stop adsClient wait for send/receive goroutine graceful exit
func (adsClient *ADSClient) Stop() {
	adsClient.SendControlChan <- 1
	adsClient.RecvControlChan <- 1
	for i := 0; i < 2; i++ {
		select {
		case <-adsClient.StopChan:
			log.DefaultLogger.Tracef("stop signal")
		}
	}
	close(adsClient.SendControlChan)
	close(adsClient.RecvControlChan)
	close(adsClient.StopChan)
}
