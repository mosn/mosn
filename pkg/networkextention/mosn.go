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
	_ "mosn.io/mosn/pkg/admin/debug"
	_ "mosn.io/mosn/pkg/buffer"
	_ "mosn.io/mosn/pkg/filter/listener/originaldst"
	_ "mosn.io/mosn/pkg/filter/network/connectionmanager"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	_ "mosn.io/mosn/pkg/filter/network/streamproxy"
	_ "mosn.io/mosn/pkg/filter/stream/dsl"
	_ "mosn.io/mosn/pkg/filter/stream/dubbo"
	_ "mosn.io/mosn/pkg/filter/stream/faultinject"
	_ "mosn.io/mosn/pkg/filter/stream/faulttolerance"
	_ "mosn.io/mosn/pkg/filter/stream/gzip"
	_ "mosn.io/mosn/pkg/filter/stream/mirror"
	_ "mosn.io/mosn/pkg/filter/stream/mixer"
	_ "mosn.io/mosn/pkg/filter/stream/payloadlimit"
	_ "mosn.io/mosn/pkg/filter/stream/stats"
	_ "mosn.io/mosn/pkg/filter/stream/transcoder/http2bolt"
	_ "mosn.io/mosn/pkg/metrics/sink"
	_ "mosn.io/mosn/pkg/metrics/sink/prometheus"
	_ "mosn.io/mosn/pkg/network"
	_ "mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/protocol/http2/conv"
	_ "mosn.io/mosn/pkg/protocol/xprotocol"
	_ "mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	_ "mosn.io/mosn/pkg/protocol/xprotocol/boltv2"
	_ "mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	_ "mosn.io/mosn/pkg/protocol/xprotocol/tars"
	_ "mosn.io/mosn/pkg/router"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	_ "mosn.io/mosn/pkg/stream/xprotocol"
	_ "mosn.io/mosn/pkg/trace/skywalking"
	_ "mosn.io/mosn/pkg/trace/skywalking/http"
	_ "mosn.io/mosn/pkg/trace/sofa/http"
	_ "mosn.io/mosn/pkg/trace/sofa/xprotocol"
	_ "mosn.io/mosn/pkg/trace/sofa/xprotocol/bolt"
	_ "mosn.io/mosn/pkg/upstream/healthcheck"
	_ "mosn.io/mosn/pkg/upstream/servicediscovery/dubbod"
	_ "mosn.io/mosn/pkg/xds"

	_ "mosn.io/mosn/pkg/networkextention/discovery/fsdis"
	_ "mosn.io/mosn/pkg/networkextention/l7/stream/filter/echo"
	_ "mosn.io/mosn/pkg/networkextention/l7/stream/filter/metadata"
	_ "mosn.io/mosn/pkg/networkextention/l7/stream/filter/metadata/unit"
)

func main() {}
