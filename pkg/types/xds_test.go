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

package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitXdsFlags(t *testing.T) {
	InitXdsFlags("cluster", "node", []string{}, []string{})
	xdsInfo := GetGlobalXdsInfo()

	if !assert.Equal(t, "cluster", xdsInfo.ServiceCluster, "serviceCluster should be 'cluster'") {
		t.FailNow()
	}
	if !assert.Equal(t, "node", xdsInfo.ServiceNode, "serviceNode should be 'node'") {
		t.FailNow()
	}
	//if !assert.Equal(t, 3, len(xdsInfo.Metadata.GetFields()), "serviceMeta len default be three") {
	//	t.FailNow()
	//}

	InitXdsFlags("cluster", "node", []string{
		"IstioVersion:1.1",
		"Not_exist_key:1",
		"not_exist_value",
	}, []string{})
	if !assert.Equal(t, 3, len(xdsInfo.Metadata.GetFields()), "serviceMeta len should be one") {
		t.FailNow()
	}
	if !assert.Equal(t, "1.1", xdsInfo.Metadata.Fields["ISTIO_VERSION"].GetStringValue(), "serviceMeta len should be one") {
		t.FailNow()
	}

}
