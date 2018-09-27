// Copyright (c) 2015 Asim Aslam.
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

package registry

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

import (
	jerrors "github.com/juju/errors"
)

//////////////////////////////////////////
// service url event type
//////////////////////////////////////////

type ServiceURLEventType int

const (
	ServiceURLAdd = iota
	ServiceURLDel
	ServiceURLUpdate
)

var serviceURLEventTypeStrings = [...]string{
	"add service url",
	"delete service url",
	"update service url",
}

func (t ServiceURLEventType) String() string {
	return serviceURLEventTypeStrings[t]
}

//////////////////////////////////////////
// service url event
//////////////////////////////////////////

type ServiceURLEvent struct {
	Action  ServiceURLEventType
	Service *ServiceURL
}

func (e ServiceURLEvent) String() string {
	return fmt.Sprintf("ServiceURLEvent{Action{%s}, Service{%s}}", e.Action.String(), e.Service)
}

//////////////////////////////////////////
// service url
//////////////////////////////////////////

type ServiceURL struct {
	Protocol     string
	Location     string // ip+port
	Path         string // like  /com.ikurento.dubbo.UserProvider3
	Ip           string
	Port         string
	Version      string
	Group        string
	Query        url.Values
	Weight       int32
	PrimitiveURL string
}

func (s ServiceURL) String() string {
	return fmt.Sprintf(
		"ServiceURL{Protocol:%s, Location:%s, Path:%s, Ip:%s, Port:%s, "+
			"Version:%s, Group:%s, Weight:%d, Query:%+v}",
		s.Protocol, s.Location, s.Path, s.Ip, s.Port,
		s.Version, s.Group, s.Weight, s.Query)
}

func NewServiceURL(urlString string) (*ServiceURL, error) {
	var (
		err          error
		rawUrlString string
		serviceUrl   *url.URL
		s            = &ServiceURL{}
	)

	rawUrlString, err = url.QueryUnescape(urlString)
	if err != nil {
		return nil, jerrors.Errorf("url.QueryUnescape(%s),  error{%v}", urlString, err)
	}

	serviceUrl, err = url.Parse(rawUrlString)
	if err != nil {
		return nil, jerrors.Errorf("url.Parse(url string{%s}),  error{%v}", rawUrlString, err)
	}

	s.Query, err = url.ParseQuery(serviceUrl.RawQuery)
	if err != nil {
		return nil, jerrors.Errorf("url.ParseQuery(raw url string{%s}),  error{%v}", serviceUrl.RawQuery, err)
	}

	s.PrimitiveURL = urlString
	s.Protocol = serviceUrl.Scheme
	s.Location = serviceUrl.Host
	s.Path = serviceUrl.Path
	if strings.Contains(s.Location, ":") {
		s.Ip, s.Port, err = net.SplitHostPort(s.Location)
		if err != nil {
			return nil, jerrors.Errorf("net.SplitHostPort(Url.Host{%s}), error{%v}", s.Location, err)
		}
	}
	s.Group = s.Query.Get("group")
	s.Version = s.Query.Get("version")

	return s, nil
}

func (s *ServiceURL) ServiceConfig() ServiceConfig {
	interfaceName := s.Query.Get("interface")
	return ServiceConfig{
		Protocol: s.Protocol,
		Service:  interfaceName,
		Group:    s.Group,
		Version:  s.Version,
	}
}

func (s *ServiceURL) CheckMethod(method string) bool {
	var (
		methodArray []string
	)

	methodArray = strings.Split(s.Query.Get("methods"), ",")
	for _, m := range methodArray {
		if m == method {
			return true
		}
	}

	return false
}
