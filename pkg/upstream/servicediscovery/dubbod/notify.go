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

package dubbod

import (
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	clusterAdapter "mosn.io/mosn/pkg/upstream/cluster"
	registry "mosn.io/pkg/registry/dubbo"
	"mosn.io/pkg/registry/dubbo/remoting"
)

// listener listens for registry subscription data change
type listener struct{}

func (l *listener) Notify(event *registry.ServiceEvent) {
	var (
		err         error
		clusterName = event.Service.Service() // FIXME
		addr        = event.Service.Ip + ":" + event.Service.Port
	)

	switch event.Action {
	case remoting.EventTypeAdd:
		err = clusterAdapter.GetClusterMngAdapterInstance().TriggerHostAppend(clusterName, []v2.Host{
			{
				HostConfig: v2.HostConfig{
					Address: addr,
				},
			},
		})
		if err != nil {
			// call the cluster manager to add the host
			err = clusterAdapter.GetClusterMngAdapterInstance().TriggerClusterAndHostsAddOrUpdate(
				v2.Cluster{
					Name:                 clusterName,
					ClusterType:          v2.SIMPLE_CLUSTER,
					LbType:               v2.LB_RANDOM,
					MaxRequestPerConn:    1024,
					ConnBufferLimitBytes: 32768,
				},
				[]v2.Host{
					{
						HostConfig: v2.HostConfig{
							Address: addr,
						},
					},
				},
			)
		}
	case remoting.EventTypeDel:
		// call the cluster manager to del the host
		err = clusterAdapter.GetClusterMngAdapterInstance().TriggerHostDel(clusterName, []string{addr})
	case remoting.EventTypeUpdate:
		fallthrough
	default:
		log.DefaultLogger.Warnf("not supported")
	}

	if err != nil {
		log.DefaultLogger.Errorf("process zk event fail, err: %v", err.Error())
	}
}
