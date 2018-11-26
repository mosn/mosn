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

package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// Mock to test shadow stream
// test shadow upstream response, then release the shadow downstream
func TestShadowStream(t *testing.T) {
	proxy := &proxy{
		context:        context.WithValue(context.Background(), types.ContextKeyLogger, log.DefaultLogger),
		clusterManager: &mockClusterManager{},
		config: &v2.Proxy{
			UpstreamProtocol:   string(protocol.HTTP1),
			DownstreamProtocol: string(protocol.HTTP1),
		},
	}
	originalDownstream := &downStream{
		proxy:                 proxy,
		streamID:              "1",
		route:                 &mockRoute{},
		downstreamReqHeaders:  protocol.CommonHeader{},
		downstreamReqDataBuf:  buffer.NewIoBuffer(0),
		downstreamReqTrailers: protocol.CommonHeader{},
	}
	shadowStream := newShadowDownstream(originalDownstream)
	if shadowStream == nil {
		t.Fatal("create shadow stream failed")
	}
	shadowStream.doShadow()
	// wait event process done
	time.Sleep(time.Second)
	if !(shadowStream.proxy == nil &&
		shadowStream.route == nil &&
		shadowStream.cluster == nil &&
		shadowStream.upstreamRequest == nil &&
		shadowStream.timeout == nil &&
		shadowStream.reqTimer == nil &&
		shadowStream.downstreamReqHeaders == nil &&
		shadowStream.downstreamReqDataBuf == nil &&
		shadowStream.downstreamReqTrailers == nil &&
		shadowStream.context == nil &&
		shadowStream.logger == nil &&
		shadowStream.snapshot == nil) {
		t.Errorf("shadow stream is not release %v", shadowStream)
	}
}
