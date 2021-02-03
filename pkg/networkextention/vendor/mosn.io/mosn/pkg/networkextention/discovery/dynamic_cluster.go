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

package discovery

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/utils"
)

//func init() {
//	if l, err := NewMonitorDiscovery("mockdiscovery"); err == nil {
//		l.start()
//	} else {
//		log.DefaultLogger.Fatalf("NewMonitorDiscovery failed: %v", err)
//	}
//}

const (
	defailtEnvoyAdminAPIUrl4UpdateCluster = "http://localhost:3491/addorupdatecluster"
	defailtEnvoyAdminAPIUrl4CheckReady    = "http://localhost:3491/ready"
)

const (
	clusterTemplateName = "clusterTemplateName"
)

const (
	DiscoveryLoopInterval       = 3 * time.Second
	DiscoveryLoopInterval4RETRY = 1 * time.Second
	maxTimes                    = 5
)

// subset cluster information, TODO add a subsetCluster struct.
const subsetClusterStart = `{
"name": "` + clusterTemplateName + `",
"connect_timeout": "3s",
"type": "STRICT_DNS",
"dns_lookup_family": "V4_ONLY",
"lb_policy": "ROUND_ROBIN",
"http2_protocol_options": {
"max_concurrent_streams": 40960,
"initial_stream_window_size": 65536,
"initial_connection_window_size": 1048576
},
"lb_subset_config": {
"fallback_policy": "ANY_ENDPOINT",
"subset_selectors": [
  {
    "keys": [
      "idc"
    ],
    "fallback_policy": "ANY_ENDPOINT"
  }
]
},
"load_assignment": {
"cluster_name": "` + clusterTemplateName + `",
"endpoints": [
  {
    "lb_endpoints":  `

const subsetClusterEnd = `
  }
]
},
"circuit_breakers": {
"thresholds": [
  {
    "priority": "DEFAULT",
    "max_connections": 1000000000,
    "max_pending_requests": 1000000000,
    "max_requests": 1000000000
  },
  {
    "priority": "HIGH",
    "max_connections": 1000000000,
    "max_pending_requests": 1000000000,
    "max_requests": 1000000000
  }]}}`

// hostAndMetadata is a endpoint struct for creating subset cluster.
type hostAndMetadata struct {
	Endpoint struct {
		Address struct {
			SocketAddress struct {
				Address   string `json:"address"`
				PortValue int    `json:"port_value"`
			} `json:"socket_address"`
		} `json:"address"`
	} `json:"endpoint"`
	Metadata struct {
		FilterMetadata struct {
			EnvoyLb struct {
				Idc string `json:"idc"`
			} `json:"envoy.lb"`
		} `json:"filter_metadata"`
	} `json:"metadata"`
}

type MonitorDiscovery struct {
	wg        sync.WaitGroup
	done      int32
	discovery Discovery
}

// NewMonitorDiscovery is used to create a discovery instance with the name 'typ' for add or update cluster
func NewMonitorDiscovery(typ string) (*MonitorDiscovery, error) {
	md := &MonitorDiscovery{}

	if disy, err := GetDiscovery(typ); err != nil {
		return nil, err
	} else {
		md.discovery = disy
		return md, nil
	}
}

func (l *MonitorDiscovery) Start() {
	l.wg.Add(1)
	utils.GoWithRecover(func() {
		defer l.wg.Done()

		times := 0
		updateSuc := false
		duration := int(DiscoveryLoopInterval)
		var endpointer vipTransferEndpoints
		endpointer.InitEndpoint(10)

		for {
			if l.IsClosed() {
				break
			}

			time.Sleep(time.Duration(times * duration))
			if !updateSuc {
				times = 1
			} else {
				times++
			}

			if times > maxTimes {
				times = maxTimes
			}

			services := l.discovery.GetAllServiceName()
			if len(services) == 0 {
				log.DefaultLogger.Warnf("[dynamiccluster][start] get empty services")
				continue
			}

			for _, service := range services {
				if changed := l.discovery.CheckAndResetServiceChange(service); changed {
					servicesAddrInfo := l.discovery.GetServiceAddrInfo(service)
					for _, si := range servicesAddrInfo {
						endpointer.AddEndpoint(si.GetServiceIP(), si.GetServiceSite(), si.GetServicePort())
					}

					serviceLength := endpointer.GetEndPointsLength()
					if serviceLength == 0 {
						log.DefaultLogger.Warnf("[dynamiccluster][start] get empty server list from service %+v", service)
						continue
					}

					if cs := endpointer.GenerateHackCluster(service); len(cs) != 0 {
						if err := notifyEnvoyAddorUpdateCluster(cs); err != nil {
							updateSuc = false
							times = 1
							duration = int(DiscoveryLoopInterval4RETRY)
							l.discovery.SetServiceChangeRetry(service)
							log.DefaultLogger.Errorf("[dynamiccluster][start] notifyEnvoyAddorUpdateCluster failed: %v", err)
						} else {
							duration = int(DiscoveryLoopInterval)
							updateSuc = true
							log.DefaultLogger.Warnf("[dynamiccluster][start] notifyEnvoyAddorUpdateCluster successed, serviceLength: %d", serviceLength)
						}

					}

					endpointer.Reset()
				}
			}

		}
	}, nil)
}

func (l *MonitorDiscovery) IsClosed() bool {
	return atomic.LoadInt32(&l.done) == 1
}

func (l *MonitorDiscovery) Stop() {
	atomic.StoreInt32(&l.done, 1)
	l.wg.Wait()
}

type vipTransferEndpoints struct {
	endpoints []hostAndMetadata
}

func (l *vipTransferEndpoints) InitEndpoint(size int) {
	l.endpoints = make([]hostAndMetadata, size)
	l.endpoints = l.endpoints[:0]
}

func (l *vipTransferEndpoints) AddEndpoint(ip, idc string, port uint32) {
	var endpoint hostAndMetadata
	endpoint.Endpoint.Address.SocketAddress.Address = ip
	endpoint.Endpoint.Address.SocketAddress.PortValue = int(port)
	endpoint.Metadata.FilterMetadata.EnvoyLb.Idc = idc

	l.endpoints = append(l.endpoints, endpoint)
}

func (l *vipTransferEndpoints) GetEndpointsbyJsonString() (string, error) {
	content, err := json.Marshal(l.endpoints)
	if err != nil {
		log.DefaultLogger.Errorf("[dynamiccluster][GetEndpointsbyJsonString] json Marshal string failed, error: %s", err.Error())
		return "", err
	}
	return bytes2str(content), nil
}

func (l *vipTransferEndpoints) Reset() {
	l.endpoints = l.endpoints[:0]
}

func (l *vipTransferEndpoints) GetEndPointsLength() int {
	return len(l.endpoints)
}

func (l *vipTransferEndpoints) GenerateHackCluster(serviceName string) string {
	endpoints, _ := l.GetEndpointsbyJsonString()
	if len(endpoints) != 0 {
		return strings.Replace(subsetClusterStart+endpoints+subsetClusterEnd, clusterTemplateName, serviceName, -1)
	} else {
		return ""
	}
}

func checkCurrentEnvoyReady() error {
	// use nokeepalive to health check
	client := CreateHttpClient()
	resp, err := client.Get(defailtEnvoyAdminAPIUrl4CheckReady)
	if err != nil {
		return errors.New("check current envoy ready failed: " + err.Error())
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return errors.New("check current envoy ready failed: " + string(body))
	}
	return nil
}

func notifyEnvoyAddorUpdateCluster(cluster string) error {
	if err := checkCurrentEnvoyReady(); err != nil {
		return err
	}

	// use nokeepalive to update cluster
	client := CreateHttpClient()
	resp, err := client.Post(defailtEnvoyAdminAPIUrl4UpdateCluster, "text/plain", strings.NewReader(cluster))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return errors.New("add or update cluster failed: " + string(body))
	}

	return nil
}

func CreateHttpClient() http.Client {
	tr := &http.Transport{
		DisableKeepAlives: true,
	}

	return http.Client{
		Transport: tr,
		Timeout:   time.Duration(10 * time.Second),
	}
}

func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
