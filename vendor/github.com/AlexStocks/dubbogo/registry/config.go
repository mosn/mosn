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
)

import (
	"github.com/AlexStocks/goext/net"
)

type RegistryConfig struct {
	Address  []string `required:"true"`
	UserName string
	Password string
	Timeout  int `default:"5"` // unit: second
}

type ServiceConfigIf interface {
	String() string
	ServiceEqual(url *ServiceURL) bool
}

// func (c *consumerZookeeperRegistry) Register(conf ServiceConfig) 函数用到了Service
type ServiceConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	Service  string `required:"true"`                 // 其本质是dubbo.xml中的interface
	Group    string
	Version  string
}

func (c ServiceConfig) Key() string {
	return fmt.Sprintf("%s@%s", c.Service, c.Protocol)
}

func (c ServiceConfig) String() string {
	return fmt.Sprintf("%s@%s-%s-%s", c.Service, c.Protocol, c.Group, c.Version)
}

// 目前不支持一个service的多个协议的使用，将来如果要支持，关键点就是修改这个函数
func (c ServiceConfig) ServiceEqual(url *ServiceURL) bool {
	if c.Protocol != url.Protocol {
		return false
	}

	if c.Service != url.Query.Get("interface") {
		return false
	}

	//if c.Group != "" && c.Group != url.Group {
	if c.Group != url.Group {
		return false
	}

	//if c.Version != "" && c.Version != url.Version {
	if c.Version != url.Version {
		return false
	}

	return true
}

type ProviderServiceConfig struct {
	ServiceConfig
	Path    string
	Methods string
}

func (c ProviderServiceConfig) String() string {
	return fmt.Sprintf(
		"%s@%s-%s-%s-%s/%s",
		c.ServiceConfig.Service,
		c.ServiceConfig.Protocol,
		c.ServiceConfig.Group,
		c.ServiceConfig.Version,
		c.Path,
		c.Methods,
	)
}

func (c ProviderServiceConfig) ServiceEqual(url *ServiceURL) bool {
	if c.ServiceConfig.Protocol != url.Protocol {
		return false
	}

	if c.ServiceConfig.Service != url.Query.Get("interface") {
		return false
	}

	if c.Group != "" && c.ServiceConfig.Group != url.Group {
		return false
	}

	if c.ServiceConfig.Version != "" && c.ServiceConfig.Version != url.Version {
		return false
	}

	if c.Path != url.Path {
		return false
	}

	if c.Methods != url.Query.Get("methods") {
		return false
	}

	return true
}

type ServerConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	IP       string
	Port     int `required:"true"`
}

func (c *ServerConfig) Address() string {
	return gxnet.HostAddress(c.IP, c.Port)
}
