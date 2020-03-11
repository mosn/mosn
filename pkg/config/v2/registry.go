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

package v2

import "encoding/json"

// ServiceRegistryInfo
type ServiceRegistryInfo struct {
	ServiceAppInfo ApplicationInfo     `json:"application,omitempty"`
	ServicePubInfo []PublishInfo       `json:"publish_info,omitempty"`
	MsgMetaInfo    map[string][]string `json:"msg_meta_info,omitempty"`
	MqClientKey    map[string]string   `json:"mq_client_key,omitempty"`
	MqMeta         map[string]string   `json:"mq_meta_info,omitempty"`
	MqConsumers    map[string][]string `json:"mq_consumers,omitempty"`
}

type ApplicationInfo struct {
	AntShareCloud    bool   `json:"ant_share_cloud,omitempty"`
	DataCenter       string `json:"data_center,omitempty"`
	AppName          string `json:"app_name,omitempty"`
	Zone             string `json:"zone,omitempty"`
	DeployMode       bool   `json:"deploy_mode,omitempty"`
	MasterSystem     bool   `json:"master_system,omitempty"`
	CloudName        string `json:"cloud_name,omitempty"`
	HostMachine      string `json:"host_machine,omitempty"`
	InstanceId       string `json:"instance_id,omitempty"`
	RegistryEndpoint string `json:"registry_endpoint,omitempty"`
	AccessKey        string `json:"access_key,omitempty"`
	SecretKey        string `json:"secret_key,omitempty"`
}

// PublishInfo implements json.Marshaler and json.Unmarshaler
type PublishInfo struct {
	Pub PublishContent
}

func (pb PublishInfo) MarshalJSON() (b []byte, err error) {
	return json.Marshal(pb.Pub)
}
func (pb *PublishInfo) UnmarshalJSON(b []byte) (err error) {
	return json.Unmarshal(b, &pb.Pub)
}

type PublishContent struct {
	ServiceName string `json:"service_name,omitempty"`
	PubData     string `json:"pub_data,omitempty"`
}
