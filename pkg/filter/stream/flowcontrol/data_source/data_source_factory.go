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

type ResourceType string

var (
	creatorDataSourceFactory = make(map[ResourceType]DataSourceFactoryCreator)
)

type DataSourceFactoryCreator func(string) (DataSourceFactory, error)

type DynamicDataResource struct {
	ResourceType ResourceType `json:"resource_type"`
	Config       interface{}  `json:"config"`
}

type DataSourceFactory interface {
	CreateDataSource(interface{}) (DataSource, error)
}

type DataSource interface {
	InitFlowRules() error
}

func RegisterDataSourceFactoryCreator(rt ResourceType, creator DataSourceFactoryCreator) {
	creatorDataSourceFactory[rt] = creator
}

func GetDataSourceFactoryCreator(rt ResourceType) (DataSourceFactoryCreator, bool) {
	dataSourceFactoryCreator, ok := creatorDataSourceFactory[rt]
	return dataSourceFactoryCreator, ok
}
