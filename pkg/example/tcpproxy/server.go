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
package main

import (
	"fmt"

	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/cluster"

	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	go func() {
		// pprof server
		http.ListenAndServe("0.0.0.0:9090", nil)
	}()

	var srv server.Server

	go func() {
		// mesh
		cmf := &clusterManagerFilter{}
		cm := cluster.NewClusterManager(nil, nil, nil, false, false)
		srv = server.NewServer(nil, cmf, cm)
		srv.AddListener(tcpListener(), &proxy.TcpProxyFilterConfigFactory{
			Proxy: tcpProxyConfig(),
		}, nil)
		cmf.cccb.UpdateClusterConfig(clusters())
		cmf.chcb.UpdateClusterHost(TestCluster, 0, hosts("11.162.169.38:80"))

		srv.Start()
	}()

	select {
	case <-time.After(time.Second * 100):
		srv.Close()
		fmt.Println("[MAIN]closing..")
	}
}
