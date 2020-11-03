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
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	jsoniter "github.com/json-iterator/go"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/router"
	"mosn.io/mosn/pkg/server"
	clusterAdapter "mosn.io/mosn/pkg/upstream/cluster"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// ConvertXXX Function converts protobuf to mosn config, and makes the config effects

// ConvertAddOrUpdateRouters converts router configurationm, used to add or update routers
func ConvertAddOrUpdateRouters(routers []*envoy_api_v2.RouteConfiguration) {
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
}

// ConvertAddOrUpdateListeners converts listener configuration, used to  add or update listeners
func ConvertAddOrUpdateListeners(listeners []*envoy_api_v2.Listener) {
	for _, listener := range listeners {
		log.DefaultLogger.Debugf("xds convert listener config: %+v", listener)

		mosnListener := ConvertListenerConfig(listener)
		if mosnListener == nil {
			log.DefaultLogger.Errorf("xds client ConvertListenerConfig failed")
			Stats.LdsUpdateReject.Inc(1)
			continue
		}

		listenerAdapter := server.GetListenerAdapterInstance()
		if listenerAdapter == nil {
			// if listenerAdapter is nil, return directly
			log.DefaultLogger.Errorf("listenerAdapter is nil and hasn't been initiated at this time")
			Stats.LdsUpdateReject.Inc(1)
			return
		}

		log.DefaultLogger.Debugf("listenerAdapter.AddOrUpdateListener called, with mosn Listener:%+v", mosnListener)

		if err := listenerAdapter.AddOrUpdateListener("", mosnListener); err == nil {
			log.DefaultLogger.Debugf("xds AddOrUpdateListener success,listener address = %s", mosnListener.Addr.String())
			Stats.LdsUpdateSuccess.Inc(1)
		} else {
			log.DefaultLogger.Errorf("xds AddOrUpdateListener failure,listener address = %s, msg = %s ",
				mosnListener.Addr.String(), err.Error())
			Stats.LdsUpdateReject.Inc(1)
		}
	}

}

// ConvertDeleteListeners converts listener configuration, used to delete listener
func ConvertDeleteListeners(listeners []*envoy_api_v2.Listener) {
	for _, listener := range listeners {
		mosnListener := ConvertListenerConfig(listener)
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

// ConvertUpdateClusters converts cluster configuration, used to udpate cluster
func ConvertUpdateClusters(clusters []*envoy_api_v2.Cluster) {
	for _, cluster := range clusters {
		if jsonStr, err := json.Marshal(cluster); err == nil {
			log.DefaultLogger.Tracef("raw cluster config: %s", string(jsonStr))
		}
	}

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
			Stats.CdsUpdateReject.Inc(1)
		} else {
			log.DefaultLogger.Debugf("xds OnUpdateClusters success,cluster name = %s", cluster.Name)
			Stats.CdsUpdateSuccess.Inc(1)
		}
	}

}

// ConvertDeleteClusters converts cluster configuration, used to delete cluster
func ConvertDeleteClusters(clusters []*envoy_api_v2.Cluster) {
	mosnClusters := ConvertClustersConfig(clusters)

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

// ConverUpdateEndpoints converts cluster configuration, used to udpate hosts
func ConvertUpdateEndpoints(loadAssignments []*envoy_api_v2.ClusterLoadAssignment) error {
	var errGlobal error

	for _, loadAssignment := range loadAssignments {
		clusterName := loadAssignment.ClusterName

		for _, endpoints := range loadAssignment.Endpoints {
			hosts := ConvertEndpointsConfig(endpoints)
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
