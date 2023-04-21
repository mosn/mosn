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

package ipaccess

import (
	"context"
	"encoding/json"

	"github.com/golang/protobuf/ptypes/any"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/filter/stream/ipaccess/pb"
	"mosn.io/mosn/pkg/log"
)

const (
	allow = "allow"
	deny  = "deny"
)

func init() {
	api.RegisterStream(v2.IPAccess, CreateIPAccessFactory)
	api.RegisterXDSConfigHandler(v2.IPAccess, func(s *any.Any) (map[string]interface{}, error) {
		m := new(pb.IpAccess)
		if err := s.UnmarshalTo(m); err != nil { // transform `any` back to Struct
			log.DefaultLogger.Errorf("[IPACCESS] convert fault inject config error: %v", err)
			return nil, err
		}
		newMap, err := StructToMap(m)
		if err != nil {
			log.DefaultLogger.Errorf("[IPACCESS] convert config to map error: %v", err)
			return nil, err
		}
		return newMap, nil
	})
}

// StructToMap Converts a struct to a map while maintaining the json alias as keys
func StructToMap(obj interface{}) (newMap map[string]interface{}, err error) {
	data, err := json.Marshal(obj) // Convert to a json string

	if err != nil {
		return
	}

	err = json.Unmarshal(data, &newMap) // Convert to a map
	return
}

type Conf struct {
	DefaultAction string         `json:"default_action"`
	Header        string         `json:"header"`
	Ips           []*IpAndAction `json:"ips"`
}

type IpAndAction struct {
	Action string   `json:"action"`
	Addrs  []string `json:"addrs"`
}

type IPAccessFactory struct {
	Conf         *Conf
	AllowAll     bool
	IpAccessList []IPAccess
}

func CreateIPAccessFactory(confMap map[string]interface{}) (api.StreamFilterChainFactory, error) {
	conf, err := parseConfig(confMap)
	if err != nil {
		return nil, err
	}

	// build ipAccessList
	var ipAccessList []IPAccess
	for _, v := range conf.Ips {
		list, err := NewIpList(v.Addrs)
		if err != nil {
			return nil, err
		}
		var ipAccess IPAccess
		switch v.Action {
		case deny:
			ipAccess = &IPBlocklist{list}
		default:
			ipAccess = &IPAllowlist{list}
		}
		ipAccessList = append(ipAccessList, ipAccess)
	}
	return &IPAccessFactory{
		Conf:         conf,
		IpAccessList: ipAccessList,
		AllowAll:     conf.DefaultAction != deny,
	}, nil
}

func parseConfig(cfg map[string]interface{}) (*Conf, error) {
	conf := &Conf{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	return conf, nil
}
func (i *IPAccessFactory) CreateFilterChain(ctx context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {

	filter := NewIPAccessFilter(i.IpAccessList, i.Conf.Header, i.AllowAll)
	// ReceiverFilter, run the filter when receive a request from downstream
	// The FilterPhase can be BeforeRoute or AfterRoute, we use BeforeRoute in this demo
	callbacks.AddStreamReceiverFilter(filter, api.BeforeRoute)
	// SenderFilter, run the filter when receive a response from upstream
	// In the demo, we are not implement this filter type
	// callbacks.AddStreamSenderFilter(filter, api.BeforeSend)
}
