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

package conv

import (
	"fmt"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	jsoniter "github.com/json-iterator/go"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	clusterAdapter "mosn.io/mosn/pkg/upstream/cluster"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// ConvertAddOrUpdateRouters converts router configurationm, used to add or update routers
func (cvt *xdsConverter) ConvertAddOrUpdateRouters(routers []*envoy_config_route_v3.RouteConfiguration) {
	if routersMngIns := router.GetRoutersMangerInstance(); routersMngIns == nil {
		log.DefaultLogger.Errorf("xds OnAddOrUpdateRouters error: router manager in nil")
	} else {
		for _, router := range routers {
			log.DefaultLogger.Debugf("xds convert router config: %+v", router)

			mosnRouter, _ := ConvertRouterConf("", router)
			if err := routersMngIns.AddOrUpdateRouters(mosnRouter); err != nil {
				log.DefaultLogger.Errorf("xds client  routersMngIns.AddOrUpdateRouters error: %v", err)
			}
		}
	}
	EnvoyConfigUpdateRoutes(routers)
}

type routeHandler func(isRds bool, routerConfig *v2.RouterConfiguration)

// listenerRouterHandler handles router config in listener
func (cvt *xdsConverter) listenerRouterHandler(isRds bool, routerConfig *v2.RouterConfiguration) {
	if routerConfig == nil {
		return
	}
	// save rds records, get router config from rds request
	if isRds {
		cvt.AppendRouterName(routerConfig.RouterConfigName)
	}
	routersMngIns := router.GetRoutersMangerInstance()
	if routersMngIns == nil {
		log.DefaultLogger.Errorf("[xds] [router handler] AddOrUpdateRouters error: router manager in nil")
		return
	}
	if err := routersMngIns.AddOrUpdateRouters(routerConfig); err != nil {
		log.DefaultLogger.Errorf("[xds] [router handler]  AddOrUpdateRouters error: %v", err)
	}

}

// ConvertAddOrUpdateListeners converts listener configuration, used to  add or update listeners
func (cvt *xdsConverter) ConvertAddOrUpdateListeners(listeners []*envoy_config_listener_v3.Listener) {
	listenerAdapter := server.GetListenerAdapterInstance()
	if listenerAdapter == nil {
		// if listenerAdapter is nil, return directly
		log.DefaultLogger.Errorf("listenerAdapter is nil and hasn't been initiated at this time")
		cvt.stats.LdsUpdateReject.Inc(1)
		return
	}
	for _, listener := range listeners {
		log.DefaultLogger.Debugf("xds convert listener config: %+v", listener)
		mosnListeners := ConvertListenerConfig(listener, cvt.listenerRouterHandler)
		if len(mosnListeners) == 0 {
			log.DefaultLogger.Errorf("xds client ConvertListenerConfig failed")
			cvt.stats.LdsUpdateReject.Inc(1)
			continue // Maybe next listener is ok
		}
		for _, mosnListener := range mosnListeners {
			log.DefaultLogger.Debugf("listenerAdapter.AddOrUpdateListener called, with mosn Listener:%+v", mosnListener)
			if err := listenerAdapter.AddOrUpdateListener("", mosnListener); err == nil {
				log.DefaultLogger.Debugf("xds AddOrUpdateListener success,listener address = %s", mosnListener.Addr.String())
				cvt.stats.LdsUpdateSuccess.Inc(1)
			} else {
				log.DefaultLogger.Errorf("xds AddOrUpdateListener failure,listener address = %s, msg = %s ",
					mosnListener.Addr.String(), err.Error())
				cvt.stats.LdsUpdateReject.Inc(1)
			}
		}
	}
	EnvoyConfigUpdateListeners(listeners)
}

// ConvertDeleteListeners converts listener configuration, used to delete listener
func (cvt *xdsConverter) ConvertDeleteListeners(listeners []*envoy_config_listener_v3.Listener) {
	listenerAdapter := server.GetListenerAdapterInstance()
	if listenerAdapter == nil {
		// if listenerAdapter is nil, return directly
		log.DefaultLogger.Errorf("listenerAdapter is nil and hasn't been initiated at this time")
		cvt.stats.LdsUpdateReject.Inc(1)
		return
	}
	for _, listener := range listeners {
		mosnListeners := ConvertListenerConfig(listener, cvt.listenerRouterHandler)
		for _, mosnListener := range mosnListeners {
			if err := listenerAdapter.DeleteListener("", mosnListener.Name); err == nil {
				log.DefaultLogger.Debugf("xds OnDeleteListeners success,listener address = %s", mosnListener.Addr.String())
			} else {
				log.DefaultLogger.Errorf("xds OnDeleteListeners failure,listener address = %s, mag = %s ",
					mosnListener.Addr.String(), err.Error())
			}
		}
	}

	// cannot delete by mosn listener name
	// because an envoy_config_listener_v3.Listener maybe convert to multiple mosn listeners
	// and we record the envoy_config_listener_v3.Listener
	EnvoyConfigDeleteListeners(listeners)

}

// ConvertUpdateClusters converts cluster configuration, used to update cluster
func (cvt *xdsConverter) ConvertUpdateClusters(clusters []*envoy_config_cluster_v3.Cluster) {
	mosnClusters := ConvertClustersConfig(clusters)
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
			cvt.stats.CdsUpdateReject.Inc(1)
		} else {
			log.DefaultLogger.Debugf("xds OnUpdateClusters success,cluster name = %s", cluster.Name)
			cvt.stats.CdsUpdateSuccess.Inc(1)
		}
	}
	EnvoyConfigUpdateClusters(clusters)
}

// ConvertDeleteClusters converts cluster configuration, used to delete cluster
func (cvt *xdsConverter) ConvertDeleteClusters(clusters []*envoy_config_cluster_v3.Cluster) {
	mosnClusters := ConvertClustersConfig(clusters)
	for _, cluster := range mosnClusters {
		log.DefaultLogger.Debugf("delete cluster: %+v\n", cluster)
		if cluster.ClusterType == v2.EDS_CLUSTER {
			if err := clusterAdapter.GetClusterMngAdapterInstance().TriggerClusterDel(cluster.Name); err != nil {
				log.DefaultLogger.Errorf("xds OnDeleteClusters failed,cluster name = %s, error: %v", cluster.Name, err.Error())

			} else {
				log.DefaultLogger.Debugf("xds OnDeleteClusters success,cluster name = %s", cluster.Name)
				EnvoyConfigDeleteClusterByName(cluster.Name)
			}
		}
	}
}

// ConverUpdateEndpoints converts cluster configuration, used to udpate hosts
func (cvt *xdsConverter) ConvertUpdateEndpoints(loadAssignments []*envoy_config_endpoint_v3.ClusterLoadAssignment) error {
	var errGlobal error
	clusterMngAdapter := clusterAdapter.GetClusterMngAdapterInstance()
	if clusterMngAdapter == nil {
		log.DefaultLogger.Errorf("xds client update Error: clusterMngAdapter nil")
		return fmt.Errorf("xds client update Error: clusterMngAdapter nil")
	}

	for _, loadAssignment := range loadAssignments {
		clusterName := loadAssignment.ClusterName

		if len(loadAssignment.Endpoints) == 0 {
			if err := clusterAdapter.GetClusterMngAdapterInstance().TriggerClusterHostUpdate(clusterName, nil); err != nil {
				log.DefaultLogger.Errorf("xds client update Error = %s, hosts are is empty", err.Error())
				errGlobal = fmt.Errorf("xds client update Error = %s, hosts are is empty", err.Error())
			} else {
				log.DefaultLogger.Debugf("xds client update host success,hosts is empty")
			}
			continue
		}

		for _, endpoints := range loadAssignment.Endpoints {
			hosts := ConvertEndpointsConfig(endpoints)
			log.DefaultLogger.Debugf("xds client update endpoints: cluster: %s, priority: %d", loadAssignment.ClusterName, endpoints.Priority)
			for index, host := range hosts {
				log.DefaultLogger.Debugf("host[%d] is : %+v", index, host)
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
