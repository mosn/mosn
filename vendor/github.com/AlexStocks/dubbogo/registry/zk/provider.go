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

package zookeeper

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/version"
)

const (
	ProviderRegistryZkClient = "consumer zk registry"
)

type providerZookeeperRegistry struct {
	*zookeeperRegistry
	zkPath map[string]int // key = protocol://ip:port/interface
}

func NewProviderZookeeperRegistry(opts ...registry.Option) registry.Registry {
	var (
		err     error
		options registry.Options
		reg     *zookeeperRegistry
		s       *providerZookeeperRegistry
	)

	options = registry.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	reg, err = newZookeeperRegistry(options)
	if err != nil {
		return nil
	}
	reg.client.name = ProviderRegistryZkClient
	s = &providerZookeeperRegistry{zookeeperRegistry: reg, zkPath: make(map[string]int)}
	s.wg.Add(1)
	go s.handleZkRestart()

	return s
}

func (s *providerZookeeperRegistry) validateZookeeperClient() error {
	var (
		err error
	)

	err = nil
	s.Lock()
	if s.client == nil {
		s.client, err = newZookeeperClient(ProviderRegistryZkClient, s.Address, s.RegistryConfig.Timeout)
		if err != nil {
			log.Warn("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%#v}",
				ProviderRegistryZkClient, s.Address, s.Timeout, jerrors.ErrorStack(err))
		}
	}
	s.Unlock()

	return jerrors.Annotatef(err, "newZookeeperClient(ProviderRegistryZkClient, addr:%+v)", s.Address)
}

func (s *providerZookeeperRegistry) Register(c interface{}) error {
	var (
		ok   bool
		err  error
		conf registry.ProviderServiceConfig
	)

	if conf, ok = c.(registry.ProviderServiceConfig); !ok {
		return jerrors.Errorf("@c{%v} type is not registry.ServiceConfig", c)
	}

	// 检验服务是否已经注册过
	ok = false
	s.Lock()
	// 注意此处与consumerZookeeperRegistry的差异，consumer用的是conf.Service，因为consumer要提供watch功能给selector使用
	// provider允许注册同一个service的多个group or version
	_, ok = s.services[conf.String()]
	s.Unlock()
	if ok {
		return jerrors.Errorf("Service{%s} has been registered", conf.String())
	}

	err = s.register(&conf)
	if err != nil {
		return jerrors.Annotatef(err, "register(conf:%+v)", conf)
	}

	s.Lock()
	s.services[conf.String()] = &conf
	log.Debug("(providerZookeeperRegistry)Register(conf{%#v})", conf)
	s.Unlock()

	return nil
}

func (s *providerZookeeperRegistry) register(conf *registry.ProviderServiceConfig) error {
	var (
		err        error
		revision   string
		params     url.Values
		urlPath    string
		rawURL     string
		encodedURL string
		dubboPath  string
	)

	if conf.ServiceConfig.Service == "" || conf.Methods == "" {
		return jerrors.Errorf("conf{Service:%s, Methods:%s}", conf.ServiceConfig.Service, conf.Methods)
	}

	err = s.validateZookeeperClient()
	if err != nil {
		return jerrors.Trace(err)
	}
	// 先创建服务下面的provider node
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, DubboNodes[PROVIDER])
	s.Lock()
	err = s.client.Create(dubboPath)
	s.Unlock()
	if err != nil {
		log.Error("zkClient.create(path{%s}) = error{%#v}", dubboPath, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "zkclient.Create(path:%s)", dubboPath)
	}

	params = url.Values{}
	params.Add("interface", conf.ServiceConfig.Service)
	params.Add("application", s.ApplicationConfig.Name)
	revision = s.ApplicationConfig.Version
	if revision == "" {
		revision = "0.1.0"
	}
	params.Add("revision", revision) // revision是pox.xml中application的version属性的值
	if conf.ServiceConfig.Group != "" {
		params.Add("group", conf.ServiceConfig.Group)
	}
	// dubbo java consumer来启动找provider url时，因为category不匹配，会找不到provider，导致consumer启动不了,所以使用consumers&providers
	// DubboRole               = [...]string{"consumer", "", "", "provider"}
	// params.Add("category", (DubboType(PROVIDER)).Role())
	params.Add("category", (DubboType(PROVIDER)).String())
	params.Add("dubbo", "dubbo-provider-golang-"+version.Version)
	params.Add("org", s.ApplicationConfig.Organization)
	params.Add("module", s.ApplicationConfig.Module)
	params.Add("owner", s.ApplicationConfig.Owner)
	params.Add("side", (DubboType(PROVIDER)).Role())
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("timeout", fmt.Sprintf("%v", s.Timeout))
	// params.Add("timestamp", time.Now().Format("20060102150405"))
	params.Add("timestamp", fmt.Sprintf("%d", s.birth))
	if conf.ServiceConfig.Version != "" {
		params.Add("version", conf.ServiceConfig.Version)
	}
	if conf.Methods != "" {
		params.Add("methods", conf.Methods)
	}
	log.Debug("provider zk url params:%#v", params)
	if conf.Path == "" {
		conf.Path = localIP
	}

	urlPath = conf.Service
	if s.zkPath[urlPath] != 0 {
		urlPath += strconv.Itoa(s.zkPath[urlPath])
	}
	s.zkPath[urlPath]++
	rawURL = fmt.Sprintf("%s://%s/%s?%s", conf.Protocol, conf.Path, urlPath, params.Encode())
	encodedURL = url.QueryEscape(rawURL)

	// 把自己注册service providers
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, (DubboType(PROVIDER)).String())
	err = s.registerTempZookeeperNode(dubboPath, encodedURL)
	log.Debug("provider path:%s, url:%s", dubboPath, rawURL)
	if err != nil {
		return jerrors.Annotatef(err, "registerTempZookeeperNode(path:%s, url:%s)", dubboPath, rawURL)
	}

	return nil
}

func (s *providerZookeeperRegistry) handleZkRestart() {
	var (
		err       error
		flag      bool
		failTimes int
		confIf    registry.ServiceConfigIf
		services  []registry.ServiceConfigIf
	)

	defer s.wg.Done()
LOOP:
	for {
		select {
		case <-s.done:
			log.Warn("(providerZookeeperRegistry)reconnectZkRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-s.client.done():
			s.Lock()
			s.client.Close()
			s.client = nil
			s.Unlock()

			// 接zk，直至成功
			failTimes = 0
			for {
				select {
				case <-s.done:
					log.Warn("(providerZookeeperRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-time.After(common.TimeSecondDuration(failTimes * registry.REGISTRY_CONN_DELAY)): // 防止疯狂重连zk
				}
				err = s.validateZookeeperClient()
				log.Info("providerZookeeperRegistry.validateZookeeperClient(zkAddr{%s}) = error{%#v}",
					s.client.zkAddrs, jerrors.ErrorStack(err))
				if err == nil {
					// copy s.services
					s.Lock()
					for _, confIf = range s.services {
						services = append(services, confIf)
					}
					s.Unlock()

					flag = true
					for _, confIf = range services {
						err = s.register(confIf.(*registry.ProviderServiceConfig))
						if err != nil {
							log.Error("(providerZookeeperRegistry)register(conf{%#v}) = error{%#v}",
								confIf.(*registry.ProviderServiceConfig), jerrors.ErrorStack(err))
							flag = false
							break
						}
					}
					if flag {
						break
					}
				}
				failTimes++
				if MAX_TIMES <= failTimes {
					failTimes = MAX_TIMES
				}
			}
		}
	}
}

func (s *providerZookeeperRegistry) String() string {
	return "dubbogo-provider-zookeeper-registry"
}

func (s *providerZookeeperRegistry) closeRegisters() {
	s.Lock()
	defer s.Unlock()
	log.Info("begin to close provider zk client")
	// 先关闭旧client，以关闭tmp node
	s.client.Close()
	s.client = nil
	//for key = range s.services {
	//	log.Debug("delete register provider zk path:%s", key)
	//	// 	delete(s.services, key)
	//}
	s.services = nil
}

func (r *providerZookeeperRegistry) GetServices(registry.ServiceConfigIf) ([]*registry.ServiceURL, error) {
	return nil, nil
}

func (r *providerZookeeperRegistry) Watch() (registry.Watcher, error) {
	return nil, nil
}

func (s *providerZookeeperRegistry) Close() {
	close(s.done)
	s.wg.Wait()
	s.closeRegisters()
}
