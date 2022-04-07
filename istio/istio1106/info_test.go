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

package istio1106

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/istio"
)

func TestIsApplicationNodeType(t *testing.T) {

	require.True(t, IsApplicationNodeType("sidecar"))

	require.False(t, IsApplicationNodeType("any"))
}

const xdsConfig = `{
	"id": "sidecar~172.31.1.5~sleep-64d7d56698-6z8zw.default~default.svc.cluster.local",
	"cluster": "sleep.default",
	"locality": {},
	"metadata": {"APP_CONTAINERS":"sleep","CLUSTER_ID":"Kubernetes","INSTANCE_IPS":"172.31.1.5","INTERCEPTION_MODE":"REDIRECT","ISTIO_PROXY_SHA":"istio-proxy:298ff36b2d43794816f7d8cdc5461bf6eed71bba","ISTIO_VERSION":"1.9.0","LABELS":{"app":"sleep","istio.io/rev":"default","pod-template-hash":"64d7d56698","security.istio.io/tlsMode":"istio","service.istio.io/canonical-name":"sleep","service.istio.io/canonical-revision":"latest"},"MESH_ID":"cluster.local","NAME":"sleep-64d7d56698-6z8zw","NAMESPACE":"default","OWNER":"kubernetes://apis/apps/v1/namespaces/default/deployments/sleep","POD_PORTS":"[\n]","PROXY_CONFIG":{"binaryPath":"/usr/local/bin/envoy","concurrency":2,"configPath":"./etc/istio/proxy","controlPlaneAuthPolicy":"MUTUAL_TLS","discoveryAddress":"istiod.istio-system.svc:15012","drainDuration":"45s","parentShutdownDuration":"60s","proxyAdminPort":15000,"serviceCluster":"sleep.default","statNameLength":189,"statusPort":15020,"terminationDrainDuration":"5s","tracing":{"zipkin":{"address":"zipkin.istio-system:9411"}}},"SERVICE_ACCOUNT":"sleep","WORKLOAD_NAME":"sleep"}
}`

func TestInitXdsInfo(t *testing.T) {
	cfg := &v2.MOSNConfig{
		Node: json.RawMessage(xdsConfig),
	}
	t.Run("config only", func(t *testing.T) {
		InitXdsInfo(cfg, "", "", nil, nil)
		xdsInfo := istio.GetGlobalXdsInfo()

		require.Equal(t, "sleep.default", xdsInfo.ServiceCluster, "serviceCluster should be sleep.default")
		require.Equal(t, "sidecar~172.31.1.5~sleep-64d7d56698-6z8zw.default~default.svc.cluster.local", xdsInfo.ServiceNode, "serviceNode unexpected")

		require.Len(t, xdsInfo.Metadata.GetFields(), 15)
	})

	t.Run("parameters overwrite", func(t *testing.T) {
		InitXdsInfo(cfg, "test-cluster", "test-node", []string{
			"IstioVersion:1.1",
			"Not_exist_key:1",
			"not_exist_value",
		}, []string{})

		xdsInfo := istio.GetGlobalXdsInfo()

		require.Equal(t, "test-cluster", xdsInfo.ServiceCluster)
		require.Equal(t, "test-node", xdsInfo.ServiceNode)
		require.Len(t, xdsInfo.Metadata.GetFields(), 4)
		require.Equal(t, "1.1", xdsInfo.Metadata.Fields["ISTIO_VERSION"].GetStringValue(), "serviceMeta len should be one")
		require.Equal(t, "Kubernetes", xdsInfo.Metadata.Fields["CLUSTER_ID"].GetStringValue(), "serviceMeta default specifying network is not Kubernetes")
	})
}
