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
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/server/config/proxy"
	"github.com/alipay/sofa-mosn/pkg/types"
	clusterAdapter "github.com/alipay/sofa-mosn/pkg/upstream/cluster"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// OnAddOrUpdateListeners called by XdsClient when listeners config refresh
func (config *MOSNConfig) OnAddOrUpdateListeners(listeners []*pb.Listener) error {
	for _, listener := range listeners {
		mosnListener := convertListenerConfig(listener)
		if mosnListener == nil {
			continue
		}

		var networkFilter *proxy.GenericProxyFilterConfigFactory
		var streamFilters []types.StreamFilterChainFactory

		if !mosnListener.HandOffRestoredDestinationConnections {
			for _, filterChain := range mosnListener.FilterChains {
				for _, filter := range filterChain.Filters {
					if filter.Name == v2.DEFAULT_NETWORK_FILTER {
						networkFilter = &proxy.GenericProxyFilterConfigFactory{
							Proxy: ParseProxyFilterJSON(&filter),
						}
					}
				}
			}

			streamFilters = GetStreamFilters(mosnListener.StreamFilters)

			if networkFilter == nil {
				errMsg := "xds client update listener error: proxy needed in network filters"
				log.DefaultLogger.Errorf(errMsg)
				return fmt.Errorf(errMsg)
			}
		}

		if listenerAdapter := server.GetListenerAdapterInstance(); listenerAdapter == nil {
			return fmt.Errorf("listenerAdapter is nil and hasn't been initiated at this time")
		} else {
			if err := listenerAdapter.AddOrUpdateListener("", mosnListener, networkFilter, streamFilters); err == nil {
				log.DefaultLogger.Debugf("xds AddOrUpdateListener success,listener address = %s", mosnListener.Addr.String())
			} else {
				log.DefaultLogger.Errorf("xds AddOrUpdateListener failure,listener address = %s, mag = %s ",
					mosnListener.Addr.String(), err.Error())
				return err
			}
		}
	}

	return nil
}

func (config *MOSNConfig) OnDeleteListeners(listeners []*pb.Listener) error {
	for _, listener := range listeners {
		mosnListener := convertListenerConfig(listener)
		if mosnListener == nil {
			continue
		}

		if listenerAdapter := server.GetListenerAdapterInstance(); listenerAdapter == nil {
			return fmt.Errorf("listenerAdapter is nil and hasn't been initiated at this time")
		} else {
			if err := listenerAdapter.DeleteListener("", *mosnListener); err == nil {
				log.DefaultLogger.Debugf("xds OnDeleteListeners success,listener address = %s", mosnListener.Addr.String())
			} else {
				log.DefaultLogger.Errorf("xds OnDeleteListeners failure,listener address = %s, mag = %s ",
					mosnListener.Addr.String(), err.Error())
				return err
			}
		}
	}

	return nil
}

// OnUpdateClusters called by XdsClient when clusters config refresh
func (config *MOSNConfig) OnUpdateClusters(clusters []*pb.Cluster) error {
	mosnClusters := convertClustersConfig(clusters)

	for _, cluster := range mosnClusters {
		log.DefaultLogger.Debugf("cluster: %+v\n", cluster)
		var err error
		if cluster.ClusterType == v2.EDS_CLUSTER {
			err = clusterAdapter.Adap.TriggerClusterAddedOrUpdate(*cluster)
		} else {
			err = clusterAdapter.Adap.TriggerClusterAndHostsAddedOrUpdate(*cluster, cluster.Hosts)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// OnUpdateEndpoints called by XdsClient when ClusterLoadAssignment config refresh
func (config *MOSNConfig) OnUpdateEndpoints(loadAssignments []*pb.ClusterLoadAssignment) error {

	for _, loadAssignment := range loadAssignments {
		clusterName := loadAssignment.ClusterName

		for _, endpoints := range loadAssignment.Endpoints {
			hosts := convertEndpointsConfig(&endpoints)

			for _, host := range hosts {
				log.DefaultLogger.Debugf("xds client update endpoint: cluster: %s, priority: %d, %+v\n", loadAssignment.ClusterName, endpoints.Priority, host)
			}

			if err := clusterAdapter.Adap.TriggerClusterHostUpdate(clusterName, hosts); err != nil {
				log.DefaultLogger.Errorf("xds client update Error = %s", err.Error())
				return err
			}
			log.DefaultLogger.Debugf("xds client update host success")

		}
	}

	return nil
}
