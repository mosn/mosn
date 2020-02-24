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

//func TestInitXdsFlags(t *testing.T) {
//	InitXdsFlags("cluster", "node", []string{}, []string{})
//	xdsInfo := GetGlobalXdsInfo()
//
//	if !assert.Equal(t, "cluster", xdsInfo.ServiceCluster, "serviceCluster should be 'cluster'") {
//		t.FailNow()
//	}
//	if !assert.Equal(t, "node", xdsInfo.ServiceNode, "serviceNode should be 'node'") {
//		t.FailNow()
//	}
//	if !assert.Equal(t, 0, len(xdsInfo.Metadata.GetFields()), "serviceMeta len should be zero") {
//		t.FailNow()
//	}
//
//	InitXdsFlags("cluster", "node", []string{
//		"k:v",
//		"not_exist_key",
//		"not_exist_value",
//	}, []string{})
//	if !assert.Equal(t, 1, len(xdsInfo.Metadata.GetFields()), "serviceMeta len should be one") {
//		t.FailNow()
//	}
//	for k, v := range xdsInfo.Metadata.Fields {
//		if !assert.Equal(t, "k", k, "key should be 'k'") {
//			t.Fatalf("serviceMeta len should be zero")
//		}
//
//		if vv, ok := v.Kind.(*_struct.Value_StructValue); !ok {
//			t.Fatal("value should be convert to types.Value_StringValue")
//		} else {
//			if !assert.Equal(t, "v", vv.StructValue, "value should be 'v'") {
//				t.FailNow()
//			}
//		}
//	}
//
//}
