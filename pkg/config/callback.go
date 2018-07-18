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

package config

import (
	"errors"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/server/config/proxy"
	"github.com/alipay/sofa-mosn/pkg/types"
	clusterAdapter "github.com/alipay/sofa-mosn/pkg/upstream/cluster"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

func SetGlobalStreamFilter(globalStreamFilters []types.StreamFilterChainFactory) {
	if streamFilter == nil {
		streamFilter = globalStreamFilters
	}
}

// todo , no hack
var streamFilter []types.StreamFilterChainFactory

func (config *MOSNConfig) OnUpdateListeners(listeners []*pb.Listener) error {
	for _, listener := range listeners {
		mosnListener := convertListenerConfig(listener)
		if mosnListener == nil {
			continue
		}

		var networkFilter *proxy.GenericProxyFilterConfigFactory

		if !mosnListener.HandOffRestoredDestinationConnections {
			for _, filterChain := range mosnListener.FilterChains {
				for _, filter := range filterChain.Filters {
					if filter.Name == v2.DEFAULT_NETWORK_FILTER {
						networkFilter = &proxy.GenericProxyFilterConfigFactory{
							Proxy: ParseProxyFilterJson(&filter),
						}
					}
				}
			}

			if networkFilter == nil {
				errMsg := "xds client update listener error: proxy needed in network filters"
				log.DefaultLogger.Errorf(errMsg)
				return errors.New(errMsg)
			}
		}

		if server := server.GetServer(); server == nil {
			log.DefaultLogger.Fatal("Server is nil and hasn't been initiated at this time")
		} else {
			if err := server.AddListenerAndStart(mosnListener, networkFilter, streamFilter); err == nil {
				log.DefaultLogger.Debugf("xds client update listener success,listener = %+v\n", mosnListener)
			} else {
				log.DefaultLogger.Errorf("xds client update listener error,listener = %+v\n", mosnListener)
				return err
			}
		}

	}

	return nil
}

func (config *MOSNConfig) OnUpdateClusters(clusters []*pb.Cluster) error {
	mosnClusters := convertClustersConfig(clusters)

	for _, cluster := range mosnClusters {
		log.DefaultLogger.Debugf("cluster: %+v\n", cluster)
		if err := clusterAdapter.ClusterAdap.TriggerClusterUpdate(cluster.Name, cluster.Hosts); err != nil {
			log.DefaultLogger.Errorf("xds client update cluster error ,err = %s, clustername = %s , hosts = %+v",
				err.Error(), cluster.Name, cluster.Hosts)
		} else {
			log.DefaultLogger.Debugf("xds client update cluster success, clustername = %s", cluster.Name)
		}

	}

	return nil
}

func (config *MOSNConfig) OnUpdateEndpoints(loadAssignments []*pb.ClusterLoadAssignment) error {

	for _, loadAssignment := range loadAssignments {
		clusterName := loadAssignment.ClusterName

		for _, endpoints := range loadAssignment.Endpoints {
			hosts := convertEndpointsConfig(&endpoints)

			for _, host := range hosts {
				log.DefaultLogger.Debugf("xds client update endpoint: cluster: %s, priority: %d, %+v\n", loadAssignment.ClusterName, endpoints.Priority, host)
			}

			if err := clusterAdapter.ClusterAdap.TriggerClusterUpdate(clusterName, hosts); err != nil {
				log.DefaultLogger.Errorf("xds client update Error = %s", err.Error())
			} else {
				log.DefaultLogger.Debugf("xds client update host success")

			}
		}
	}

	return nil
}
