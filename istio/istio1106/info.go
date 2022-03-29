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

package istio1106

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/golang/protobuf/jsonpb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/istio"
)

func ParseXdsInfo(raw json.RawMessage) istio.XdsInfo {
	if len(raw) <= 0 {
		return istio.XdsInfo{}
	}
	var node corev3.Node
	unmarshaler := jsonpb.Unmarshaler{AllowUnknownFields: true}
	if err := unmarshaler.Unmarshal(bytes.NewReader(raw), &node); err != nil {
		return istio.XdsInfo{}
	}
	return istio.XdsInfo{
		ServiceCluster: node.Cluster,
		ServiceNode:    node.Id,
		Metadata:       node.Metadata,
	}
}

const serviceMetaSeparator = ":"

var (
	ClusterID = "Kubernetes"

	defaultMeta = &istio.Meta{
		IstioVersion:     istio.IstioVersion,
		Labels:           map[string]string{"istio": "ingressgateway"},
		InterceptionMode: istio.InterceptionRedirect,
		ClusterID:        ClusterID,
	}
)

func InitXdsInfo(config *v2.MOSNConfig, serviceCluster, serviceNode string, serviceMeta []string, labels []string) {
	info := ParseXdsInfo(config.Node)
	// parameters is prefer than config
	if serviceCluster != "" {
		info.ServiceCluster = serviceCluster
	}
	if serviceNode != "" {
		info.ServiceNode = serviceNode
	}
	if len(serviceMeta) > 0 || len(labels) > 0 {
		metadata := &_struct.Struct{}
		// TODO: get pod labels?
		if len(labels) > 0 {
			defaultMeta.Labels = make(map[string]string, len(labels))
			for _, keyValue := range labels {
				keyValueSep := strings.SplitN(keyValue, serviceMetaSeparator, 2)
				if len(keyValueSep) != 2 {
					continue
				}
				defaultMeta.Labels[keyValueSep[0]] = keyValueSep[1]
			}
		}
		for k, v := range istio.GetPodLabels() {
			defaultMeta.Labels[k] = v
		}
		for _, keyValue := range serviceMeta {
			keyValueSep := strings.SplitN(keyValue, serviceMetaSeparator, 2)
			if len(keyValueSep) != 2 {
				continue
			}

			f := reflect.ValueOf(defaultMeta).Elem().FieldByName(keyValueSep[0])
			switch f.Kind() {
			case reflect.String:
				f.SetString(keyValueSep[1])
			case reflect.Int:
				if i, e := strconv.ParseInt(keyValueSep[1], 10, 64); e == nil {
					f.SetInt(i)
				}
			}
		}
		bs, _ := json.Marshal(defaultMeta)
		_ = jsonpb.Unmarshal(bytes.NewReader(bs), metadata)
		info.Metadata = metadata
	}
	// set value
	istio.SetServiceCluster(info.ServiceCluster)
	istio.SetServiceNode(info.ServiceNode)
	istio.SetMetadata(info.Metadata)

}

type NodeType string

const (
	// SidecarProxy type is used for sidecar proxies in the application containers
	SidecarProxy NodeType = "sidecar"

	// Router type is used for standalone proxies acting as L7/L4 routers
	Router NodeType = "router"

	// AllPortsLiteral is the string value indicating all ports
	AllPortsLiteral = "*"
)

func IsApplicationNodeType(nType string) bool {
	switch NodeType(nType) {
	case SidecarProxy, Router:
		return true
	default:
		return false
	}
}
