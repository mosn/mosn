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

package seata

// TransactionInfo transaction info config
type TransactionInfo struct {
	RequestPath string `yaml:"requestPath" json:"requestPath"`
}

// TCCResource tcc resource config
type TCCResource struct {
	PrepareRequestPath  string `yaml:"prepareRequestPath" json:"prepareRequestPath"`
	CommitRequestPath   string `yaml:"commitRequestPath" json:"commitRequestPath"`
	RollbackRequestPath string `yaml:"rollbackRequestPath" json:"rollbackRequestPath"`
}

// Seata seata config
type Seata struct {
	Name     string `yaml:"name" json:"name"`
	ConfPath string `yaml:"addressing" json:"confPath"`

	TransactionInfos []*TransactionInfo `yaml:"transactionInfos" json:"transactionInfos"`
	TCCResources     []*TCCResource     `yaml:"tccResources" json:"tccResources"`
}
