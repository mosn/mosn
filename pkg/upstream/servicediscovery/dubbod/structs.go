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

package dubbod

// for sub && unsub
type subReq struct {
	Registry struct {
		UserName string `json:"username"`
		Password string `json:"password"`
		Type     string `json:"type" binding:"eq=zookeeper"`  // only zookeeper is supported currently
		Addr     string `json:"addr" binding:"hostname_port"` // eg. 127.0.0.1:2181
		//Timeout  string `json:"timeout" binding:"required"`   // 5s
	} `json:"registry"`
	Service struct {
		Interface string   `json:"interface" binding:"required"` // eg. com.mosn.service.DemoService
		Methods   []string `json:"methods" binding:"required"`   // eg. GetUser,GetProfile,UpdateName
		//Port      string   `json:"port" binding:"max=65535,min=1"` // user service port, eg. 8080
		Group string `json:"group"` // binding:"required"`
	} `json:"service"`
}

// for pub && unpub
type pubReq struct {
	Registry struct {
		UserName string `json:"username"`
		Password string `json:"password"`
		Type     string `json:"type" binding:"eq=zookeeper"`  // only zookeeper is supported currently
		Addr     string `json:"addr" binding:"hostname_port"` // eg. 127.0.0.1:2181
		//Timeout  string `json:"timeout" binding:"required"`   // 5s
	} `json:"registry"`
	Service struct {
		Interface string   `json:"interface" binding:"required"` // eg. com.mosn.service.DemoService
		Methods   []string `json:"methods" binding:"required"`   // eg. GetUser,GetProfile,UpdateName
		// Port      string   `json:"port" binding:"numeric"`       // user service port, eg. 8080
		Group   string `json:"group"`   // binding:"required"`
		Version string `json:"version"` // eg. 1.0.3
	} `json:"service"`
}

// response struct for all requests
type resp struct {
	Errno  int    `json:"err_no"`
	ErrMsg string `json:"err_msg"`
}
