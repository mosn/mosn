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

package dubbo

import (
	"reflect"
	"testing"
)

func Test_unSerialize(t *testing.T) {
	serializeId := 2
	testdata := []byte{218, 187, 194, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 225, 5, 50, 46, 54, 46, 50, 48, 48, 99, 111, 109, 46, 97, 108, 105, 98, 97, 98, 97, 46, 98, 111, 111, 116, 46, 100, 117, 98, 98, 111, 46, 100, 101, 109, 111, 46, 99, 111, 110, 115, 117, 109, 101, 114, 46, 68, 101, 109, 111, 83, 101, 114, 118, 105, 99, 101, 5, 49, 46, 48, 46, 49, 8, 115, 97, 121, 72, 101, 108, 108, 111, 18, 76, 106, 97, 118, 97, 47, 108, 97, 110, 103, 47, 83, 116, 1, 14, 105, 110, 103, 59, 3, 120, 120, 120, 72, 4, 112, 97, 116, 104, 48, 48, 99, 111, 109, 46, 97, 108, 105, 98, 97, 98, 97, 46, 98, 111, 111, 116, 46, 100, 117, 98, 98, 111, 46, 100, 101, 109, 111, 46, 99, 111, 110, 115, 117, 109, 101, 114, 46, 68, 101, 109, 111, 83, 101, 114, 118, 105, 99, 101, 9, 105, 110, 116, 101, 114, 102, 97, 99, 101, 48, 48, 99, 111, 109, 46, 97, 108, 105, 98, 97, 98, 97, 46, 98, 111, 111, 116, 46, 100, 117, 98, 98, 111, 46, 100, 101, 109, 111, 46, 99, 111, 110, 115, 117, 109, 101, 114, 46, 68, 101, 109, 111, 83, 101, 114, 118, 105, 99, 101, 7, 118, 101, 114, 115, 105, 111, 110, 5, 49, 46, 48, 46, 49, 90}
	testdata = testdata[DUBBO_HEADER_LEN:]
	tests := []struct {
		parseCtl unserializeCtl
		want     *dubboAttr
	}{
		{unserializeCtlDubboVersion, &dubboAttr{dubboVersion: "2.6.2"}},
		{unserializeCtlPath, &dubboAttr{dubboVersion: "2.6.2", serviceName: "com.alibaba.boot.dubbo.demo.consumer.DemoService", path: "com.alibaba.boot.dubbo.demo.consumer.DemoService"}},
		{unserializeCtlVersion, &dubboAttr{dubboVersion: "2.6.2", serviceName: "com.alibaba.boot.dubbo.demo.consumer.DemoService", path: "com.alibaba.boot.dubbo.demo.consumer.DemoService", version: "1.0.1"}},
		{unserializeCtlMethod, &dubboAttr{dubboVersion: "2.6.2", serviceName: "com.alibaba.boot.dubbo.demo.consumer.DemoService", path: "com.alibaba.boot.dubbo.demo.consumer.DemoService", version: "1.0.1", methodName: "sayHello"}},

		// TODO: Here we need a test data with attachments
		// {unserializeCtlAttachments, &dubboAttr{dubboVersion: "2.6.2", serviceName: "com.alibaba.boot.dubbo.demo.consumer.DemoService", path: "com.alibaba.boot.dubbo.demo.consumer.DemoService", version: "1.0.1", methodName: "sayHello", attachments: nil}},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := unSerialize(serializeId, testdata, tt.parseCtl); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unSerialize() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
