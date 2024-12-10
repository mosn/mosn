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
	"encoding/json"
	"fmt"

	"github.com/alibaba/sentinel-golang/ext/datasource"
	"github.com/alibaba/sentinel-golang/pkg/datasource/nacos"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

const (
	NacosDataSourceType = "nacos"
)

const (
	DefaultGroup = "SENTINEL_GROUP"
	DataIDSuffix = "flow-rules"
)

type NacosDataSourceConfig struct {
	vo.NacosClientParam

	DataID string
	Group  string
}

var DefaultSetNacosDataIDAndGroup = defaultSetNacosDataIDAndGroupfunc

func defaultSetNacosDataIDAndGroupfunc(nd *NacosDataSource, config interface{}) error {
	if len(nd.config.DataID) == 0 {
		nd.config.DataID = fmt.Sprintf("%s-%s", nd.appName, DataIDSuffix)
	}
	if len(nd.config.Group) == 0 {
		nd.config.Group = DefaultGroup
	}
	return nil
}

func init() {
	RegisterDataSourceFactoryCreator(NacosDataSourceType, NacosDataSourceFactoryCreator)
}

type NacosDataSourceFactory struct {
	appName string
}

func NacosDataSourceFactoryCreator(appName string) (DataSourceFactory, error) {
	return &NacosDataSourceFactory{
		appName: appName,
	}, nil
}

func (nf *NacosDataSourceFactory) CreateDataSource(config interface{}) (DataSource, error) {
	nacosDSC := &NacosDataSourceConfig{}
	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(cfg, nacosDSC)
	if err != nil {
		return nil, err
	}
	nds := &NacosDataSource{
		appName: nf.appName,
		config:  nacosDSC,
	}
	err = DefaultSetNacosDataIDAndGroup(nds, config)
	if err != nil {
		return nil, err
	}
	return nds, nil
}

type NacosDataSource struct {
	appName string
	config  *NacosDataSourceConfig
}

func (nd *NacosDataSource) InitFlowRules() error {
	client, err := clients.NewConfigClient(nd.config.NacosClientParam)
	if err != nil {
		return err
	}

	h := datasource.NewFlowRulesHandler(datasource.FlowRuleJsonArrayParser)

	nds, err := nacos.NewNacosDataSource(client, nd.config.Group, nd.config.DataID, h)
	if err != nil {
		return err
	}

	err = nds.Initialize()
	if err != nil {
		return err
	}

	return nil
}
