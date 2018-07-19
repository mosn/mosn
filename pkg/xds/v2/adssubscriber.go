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

	"github.com/alipay/sofa-mosn/pkg/config"
	"github.com/alipay/sofa-mosn/pkg/log"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

/* Start adsClient send goroutine and receive goroutine
   send goroutine periodic request lds and cds
   receive goroutine handle response for both client request and server push
 */
func (adsClient *ADSClient) Start() {
	adsClient.StreamClient = adsClient.AdsConfig.GetStreamClient()
	adsClient.MosnConfig = &config.MOSNConfig{}
	go adsClient.sendThread()
	go adsClient.receiveThread()
}

func (adsClient *ADSClient) sendThread() {
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
			log.DefaultLogger.Tracef("send thread request lds")
			err := adsClient.V2Client.reqListeners(adsClient.StreamClient)
			if err != nil {
				log.DefaultLogger.Warnf("send thread request lds fail!auto retry next period")
			}
			log.DefaultLogger.Tracef("send thread request cds")
			err = adsClient.V2Client.reqClusters(adsClient.StreamClient)
			if err != nil {
				log.DefaultLogger.Warnf("send thread request cds fail!auto retry next period")
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
			resp, err := adsClient.StreamClient.Recv()
			if err != nil {
				log.DefaultLogger.Warnf("get resp timeout: %v", err)
				continue
			}
			typeURL := resp.TypeUrl
			if typeURL == "type.googleapis.com/envoy.api.v2.Listener" {
				log.DefaultLogger.Tracef("get lds resp,handle it")
				listeners := adsClient.V2Client.handleListersResp(resp)
				log.DefaultLogger.Infof("get %d listeners from LDS", len(listeners))
				err := adsClient.MosnConfig.OnUpdateListeners(listeners)
				if err != nil {
					log.DefaultLogger.Fatalf("fail to update listeners")
					return
				}
				log.DefaultLogger.Infof("update listeners success")
			} else if typeURL == "type.googleapis.com/envoy.api.v2.Cluster" {
				log.DefaultLogger.Tracef("get cds resp,handle it")
				clusters := adsClient.V2Client.handleClustersResp(resp)
				log.DefaultLogger.Infof("get %d clusters from CDS", len(clusters))
				err := adsClient.MosnConfig.OnUpdateClusters(clusters)
				if err != nil {
					log.DefaultLogger.Fatalf("fall to update clusters")
					return
				}
				log.DefaultLogger.Infof("update clusters success")
				clusterNames := make([]string, 0)
				for _, cluster := range clusters {
					if cluster.Type == envoy_api_v2.Cluster_EDS {
						clusterNames = append(clusterNames, cluster.Name)
					}
				}
				adsClient.V2Client.reqEndpoints(adsClient.StreamClient, clusterNames)
			} else if typeURL == "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment" {
				log.DefaultLogger.Tracef("get eds resp,handle it ")
				endpoints := adsClient.V2Client.handleEndpointesResp(resp)
				log.DefaultLogger.Tracef("get %d endpoints for cluster", len(endpoints))
				err = adsClient.MosnConfig.OnUpdateEndpoints(endpoints)
				if err != nil {
					log.DefaultLogger.Fatalf("fail to update endpoints for cluster")
					return
				}
				log.DefaultLogger.Tracef("update endpoints for cluster %s success")
			}
		}
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
