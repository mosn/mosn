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
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/server"
	"github.com/alipay/sofa-mosn/pkg/types"
	clusterAdapter "github.com/alipay/sofa-mosn/pkg/upstream/cluster"
	pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

// OnRouterUpdate used to add or update routers
func (c *MOSNConfig) OnAddOrUpdateRouters(routers []*pb.RouteConfiguration) {

	if routersMngIns := router.GetRoutersMangerInstance(); routersMngIns == nil {
		log.DefaultLogger.Errorf("xds OnAddOrUpdateRouters error: router manager in nil")
	} else {

		for _, router := range routers {
			if jsonStr, err := json.Marshal(router); err == nil {
				log.DefaultLogger.Tracef("raw router config: %s", string(jsonStr))
			}

			mosnRouter, _ := convertRouterConf("", router)
			log.DefaultLogger.Tracef("mosnRouter config: %+v", mosnRouter)
			routersMngIns.AddOrUpdateRouters(mosnRouter)
		}
	}
}

// OnAddOrUpdateListeners called by XdsClient when listeners config refresh
func (c *MOSNConfig) OnAddOrUpdateListeners(listeners []*pb.Listener) {

	for _, listener := range listeners {
		if jsonStr, err := json.Marshal(listener); err == nil {
			log.DefaultLogger.Tracef("raw listener config: %s", string(jsonStr))
		}

		mosnListener := convertListenerConfig(listener)
		if mosnListener == nil {
			continue
		}

		var streamFilters []types.StreamFilterChainFactory
		var networkFilters []types.NetworkFilterChainFactory

		if !mosnListener.HandOffRestoredDestinationConnections {
			for _, filterChain := range mosnListener.FilterChains {
				nf := GetNetworkFilters(&filterChain)
				networkFilters = append(networkFilters, nf...)
			}
			streamFilters = GetStreamFilters(mosnListener.StreamFilters)

			if len(networkFilters) == 0 {
				log.DefaultLogger.Errorf("xds client update listener error: proxy needed in network filters")
				continue
			}
		}

		listenerAdapter := server.GetListenerAdapterInstance()
		if listenerAdapter == nil {
			// if listenerAdapter is nil, return directly
			log.DefaultLogger.Errorf("listenerAdapter is nil and hasn't been initiated at this time")
		}
		log.DefaultLogger.Debugf("listenerAdapter.AddOrUpdateListener called, with mosn Listener:%+v, networkFilters:%+v, streamFilters: %+v",
			mosnListener, networkFilters, streamFilters)

		if err := listenerAdapter.AddOrUpdateListener("", mosnListener, networkFilters, streamFilters); err == nil {
			log.DefaultLogger.Debugf("xds AddOrUpdateListener success,listener address = %s", mosnListener.Addr.String())
		} else {
			log.DefaultLogger.Errorf("xds AddOrUpdateListener failure,listener address = %s, msg = %s ",
				mosnListener.Addr.String(), err.Error())
		}
	}
}

func (c *MOSNConfig) OnDeleteListeners(listeners []*pb.Listener) {
	for _, listener := range listeners {
		mosnListener := convertListenerConfig(listener)
		if mosnListener == nil {
			continue
		}

		listenerAdapter := server.GetListenerAdapterInstance()
		if listenerAdapter == nil {
			log.DefaultLogger.Errorf("listenerAdapter is nil and hasn't been initiated at this time")
			return
		}
		if err := listenerAdapter.DeleteListener("", mosnListener.Name); err == nil {
			log.DefaultLogger.Debugf("xds OnDeleteListeners success,listener address = %s", mosnListener.Addr.String())
		} else {
			log.DefaultLogger.Errorf("xds OnDeleteListeners failure,listener address = %s, mag = %s ",
				mosnListener.Addr.String(), err.Error())

		}
	}
}

// OnUpdateClusters called by XdsClient when clusters config refresh
// Can be used to update and add clusters
func (c *MOSNConfig) OnUpdateClusters(clusters []*pb.Cluster) {
	for _, cluster := range clusters {
		if jsonStr, err := json.Marshal(cluster); err == nil {
			log.DefaultLogger.Tracef("raw cluster config: %s", string(jsonStr))
		}
	}

	mosnClusters := convertClustersConfig(clusters)

	for _, cluster := range mosnClusters {
		var err error
		log.DefaultLogger.Debugf("update cluster: %+v\n", cluster)
		if cluster.ClusterType == v2.EDS_CLUSTER {
			err = clusterAdapter.GetClusterMngAdapterInstance().TriggerClusterAddOrUpdate(*cluster)
		} else {
			err = clusterAdapter.GetClusterMngAdapterInstance().TriggerClusterAndHostsAddOrUpdate(*cluster, cluster.Hosts)
		}

		if err != nil {
			log.DefaultLogger.Errorf("xds OnUpdateClusters failed,cluster name = %s, error: %v", cluster.Name, err.Error())

		} else {
			log.DefaultLogger.Debugf("xds OnUpdateClusters success,cluster name = %s", cluster.Name)
		}
	}
}

// OnDeleteClusters called by XdsClient when need to delete clusters
func (c *MOSNConfig) OnDeleteClusters(clusters []*pb.Cluster) {
	mosnClusters := convertClustersConfig(clusters)

	for _, cluster := range mosnClusters {
		log.DefaultLogger.Debugf("delete cluster: %+v\n", cluster)
		var err error
		if cluster.ClusterType == v2.EDS_CLUSTER {
			err = clusterAdapter.GetClusterMngAdapterInstance().TriggerClusterDel(cluster.Name)
		}

		if err != nil {
			log.DefaultLogger.Errorf("xds OnDeleteClusters failed,cluster name = %s, error: %v", cluster.Name, err.Error())

		} else {
			log.DefaultLogger.Debugf("xds OnDeleteClusters success,cluster name = %s", cluster.Name)
		}
	}
}

// OnUpdateEndpoints called by XdsClient when ClusterLoadAssignment config refresh
func (c *MOSNConfig) OnUpdateEndpoints(loadAssignments []*pb.ClusterLoadAssignment) error {
	var errGlobal error

	for _, loadAssignment := range loadAssignments {
		clusterName := loadAssignment.ClusterName

		for _, endpoints := range loadAssignment.Endpoints {
			hosts := convertEndpointsConfig(&endpoints)
			log.DefaultLogger.Debugf("xds client update endpoints: cluster: %s, priority: %d", loadAssignment.ClusterName, endpoints.Priority)
			for index, host := range hosts {
				log.DefaultLogger.Debugf("host[%d] is : %+v", index, host)
			}

			clusterMngAdapter := clusterAdapter.GetClusterMngAdapterInstance()
			if clusterMngAdapter == nil {
				log.DefaultLogger.Errorf("xds client update Error: clusterMngAdapter nil , hosts are %+v", hosts)
				errGlobal = fmt.Errorf("xds client update Error: clusterMngAdapter nil , hosts are %+v", hosts)
			}

			if err := clusterAdapter.GetClusterMngAdapterInstance().TriggerClusterHostUpdate(clusterName, hosts); err != nil {
				log.DefaultLogger.Errorf("xds client update Error = %s, hosts are %+v", err.Error(), hosts)
				errGlobal = fmt.Errorf("xds client update Error = %s, hosts are %+v", err.Error(), hosts)

			} else {
				log.DefaultLogger.Debugf("xds client update host success,hosts are %+v", hosts)
			}
		}
	}

	return errGlobal
}
