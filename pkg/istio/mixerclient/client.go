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

package mixerclient

import (
	"context"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/upstream/cluster"
	"google.golang.org/grpc"
	"istio.io/api/mixer/v1"
)

type MixerClient interface {
	Report(attributes *v1.Attributes)

	SendReport(request *v1.ReportRequest) *v1.ReportResponse
}

type mixerClient struct {
	reportBatch *reportBatch
	attributeCompressor *AttributeCompressor
	client v1.MixerClient
}

func NewMixerClient(reportCluster string) MixerClient {
	snapshot := cluster.GetClusterMngAdapterInstance().GetCluster(reportCluster)
	if snapshot == nil {
		log.DefaultLogger.Errorf("get mixer server cluster config error, report cluster: %s", reportCluster)
		return nil
	}

	log.DefaultLogger.Infof("snapshot: %v", snapshot.ClusterInfo().SourceAddress().String())

	mixerAddress := snapshot.PrioritySet().GetOrCreateHostSet(0).Hosts()[0].Address().String()
	conn, err := grpc.Dial(mixerAddress, grpc.WithInsecure())
	if err != nil {
		log.DefaultLogger.Errorf("grpc dial to mixserver %s error %v", mixerAddress, err)
	}
	mixClient := v1.NewMixerClient(conn)

	client := &mixerClient{
		attributeCompressor:NewAttributeCompressor(),
		client:mixClient,
	}

	client.reportBatch = newReportBatch(client.attributeCompressor, client)
	return client
}

func (c *mixerClient) Report(attributes *v1.Attributes) {
	c.reportBatch.report(attributes)
}

func (c *mixerClient) SendReport(request *v1.ReportRequest) *v1.ReportResponse {
	response, err := c.client.Report(context.Background(), request)
	if err != nil {
		log.DefaultLogger.Errorf("send report error: %v", err)
	}
	return response
}
