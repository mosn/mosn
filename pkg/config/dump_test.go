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
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func Test_AddClusterAndDump(t *testing.T) {

	log.InitDefaultLogger("", log.INFO)

	Load("../../../mosn/resource/mosn_config.json")

	//add cluster
	AddClusterConfig([]v2.Cluster{
		v2.Cluster{
			Name:        "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0@DEFAULT",
			ClusterType: v2.DYNAMIC_CLUSTER,
			LbType:      v2.LB_RANDOM,
			Spec: v2.ClusterSpecInfo{
				Subscribes: []v2.SubscribeSpec{
					v2.SubscribeSpec{
						ServiceName: "x_test_service",
					},
				},
			},
		},
	})

	//change dump path, just for test
	ConfigPath = "../../../mosn/resource/mosn_config_dump_result.json"

	Dump(true)
}

func Test_RemoveClusterAndDump(t *testing.T) {

	log.InitDefaultLogger("", log.INFO)

}

func Test_ServiceRegistryInfoDump(t *testing.T) {

	log.InitDefaultLogger("", log.INFO)

}
