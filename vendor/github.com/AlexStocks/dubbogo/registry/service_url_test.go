// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry // for go test -v

import (
	"fmt"
	"testing"
)

func TestServiceURL_NewServiceURL(t *testing.T) {
	var (
		err error
		url = "dubbo://116.211.15.190:20880/im.youni.weboa.common.service.IRegisterService?anyhost=true&application=weboa&dubbo=2.5.3&interface=im.youni.weboa.common.service.IRegisterService&methods=registerUser,exists&pid=13772&revision=1.2.2&side=provider&timestamp=1464255871323"
		// serviceURL *common.ServiceURL
		serviceURL *ServiceURL
	)

	// serviceURL, err = common.NewServiceURL(url)
	serviceURL, err = NewServiceURL(url)
	if err != nil {
		t.Errorf("ServiceUrl.Init(url{%s}) = %v", url, err)
	}

	/*
		serviceUrl{&common.ServiceURL{Protocol:"dubbo", Location:"116.211.15.190:20880", Path:"/im.youni.weboa.common.service.IRegisterService", Ip:"116.211.15.190", Port:"20880", Version:"", Group:"", Query:url.Values{"anyhost":[]string{"true"}, "application":[]string{"weboa"}, "interface":[]string{"im.youni.weboa.common.service.IRegisterService"}, "methods":[]string{"registerUser,exists"}, "pid":[]string{"13772"}, "revision":[]string{"1.2.2"}, "dubbo":[]string{"2.5.3"}, "side":[]string{"provider"}, "timestamp":[]string{"1464255871323"}}, :""}}
		serviceURL.protocol: dubbo
		serviceURL.location: 116.211.15.190:20880
		serviceURL.path: /im.youni.weboa.common.service.IRegisterService
		serviceURL.ip: 116.211.15.190
		serviceURL.port: 20880
		serviceURL.version:
		serviceURL.group:
		serviceURL.query: map[anyhost:[true] application:[weboa] interface:[im.youni.weboa.common.service.IRegisterService] methods:[registerUser,exists] pid:[13772] revision:[1.2.2] dubbo:[2.5.3] side:[provider] timestamp:[1464255871323]]
		serviceURL.query.interface: im.youni.weboa.common.service.IRegisterService
	*/
	fmt.Printf("serviceUrl{%#v}\n", serviceURL)
	fmt.Println("serviceURL.protocol:", serviceURL.Protocol)
	fmt.Println("serviceURL.location:", serviceURL.Location)
	fmt.Println("serviceURL.path:", serviceURL.Path)
	fmt.Println("serviceURL.ip:", serviceURL.Ip)
	fmt.Println("serviceURL.port:", serviceURL.Port)
	fmt.Println("serviceURL.version:", serviceURL.Version)
	fmt.Println("serviceURL.group:", serviceURL.Group)
	fmt.Println("serviceURL.query:", serviceURL.Query)
	fmt.Println("serviceURL.query.interface:", serviceURL.Query.Get("interface"))
}
