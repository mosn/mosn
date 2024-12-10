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

package datasource

import (
	"reflect"
	"testing"

	monkey "github.com/cch123/supermonkey"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

func TestNacosDataSourceFactory(t *testing.T) {
	appName := "sentinel-dashboard"
	dataSourceFactoryCreator, _ := NacosDataSourceFactoryCreator(appName)
	nacosConfig := &NacosDataSourceConfig{
		Group: "SENTINEL_GROUP",
		NacosClientParam: vo.NacosClientParam{
			ClientConfig: &constant.ClientConfig{
				NamespaceId: "public",
			},
			ServerConfigs: []constant.ServerConfig{
				{
					IpAddr: "127.0.0.1",
					Port:   8848,
				},
			},
		},
	}

	dataSource, err := dataSourceFactoryCreator.CreateDataSource(nacosConfig)
	assert.Nil(t, err)

	monkey.PatchInstanceMethod(reflect.TypeOf(dataSource), "InitFlowRules", func(d *NacosDataSource) error {
		return nil
	})

	err = dataSource.InitFlowRules()
	assert.Nil(t, err)
}
