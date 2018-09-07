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
	ConsumerRegistryZkClient string = "consumer zk registry"
	WatcherZkClient          string = "watcher zk registry"
)

type consumerZookeeperRegistry struct {
	*zookeeperRegistry
}

func NewConsumerZookeeperRegistry(opts ...registry.Option) registry.Registry {
	var (
		err     error
		options registry.Options
		reg     *zookeeperRegistry
		c       *consumerZookeeperRegistry
	)

	options = registry.Options{}
	for _, opt := range opts {
		opt(&options)
	}

	reg, err = newZookeeperRegistry(options)
	if err != nil {
		return nil
	}
	reg.client.name = ConsumerRegistryZkClient
	c = &consumerZookeeperRegistry{zookeeperRegistry: reg}
	c.wg.Add(1)
	go c.handleZkRestart()

	return c
}

func (c *consumerZookeeperRegistry) validateZookeeperClient() error {
	var (
		err error
	)

	err = nil
	c.Lock()
	if c.client == nil {
		c.client, err = newZookeeperClient(ConsumerRegistryZkClient, c.Address, c.RegistryConfig.Timeout)
		if err != nil {
			log.Warn("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%v}",
				ConsumerRegistryZkClient, c.Address, c.Timeout, err)
		}
	}
	c.Unlock()

	return err
}

func (c *consumerZookeeperRegistry) Register(sc interface{}) error {
	var (
		ok   bool
		err  error
		conf registry.ServiceConfig
	)

	if conf, ok = sc.(registry.ServiceConfig); !ok {
		return jerrors.Errorf("@c{%v} type is not registry.ServiceConfig", c)
	}

	// 检验服务是否已经注册过
	ok = false
	c.Lock()
	// 注意此处与providerZookeeperRegistry的差异，provider用的是conf.String()，因为provider无需提供watch功能给selector使用
	// consumer只允许把service的其中一个group&version提供给用户使用
	// _, ok = c.services[conf.String()]
	_, ok = c.services[conf.Key()]
	c.Unlock()
	if ok {
		return jerrors.Errorf("Service{%s} has been registered", conf.Service)
	}

	err = c.register(&conf)
	if err != nil {
		return err
	}

	c.Lock()
	// c.services[conf.String()] = &conf
	c.services[conf.Key()] = &conf
	log.Debug("(consumerZookeeperRegistry)Register(conf{%#v})", conf)
	c.Unlock()

	return nil
}

func (c *consumerZookeeperRegistry) register(conf *registry.ServiceConfig) error {
	var (
		err        error
		params     url.Values
		revision   string
		rawURL     string
		encodedURL string
		dubboPath  string
	)

	err = c.validateZookeeperClient()
	if err != nil {
		log.Error("client.validateZookeeperClient() = err:%#v", err)
		return jerrors.Trace(err)
	}
	// 创建服务下面的consumer node
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, DubboNodes[CONSUMER])
	c.Lock()
	err = c.client.Create(dubboPath)
	c.Unlock()
	if err != nil {
		log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}
	// 创建服务下面的provider node，以方便watch直接观察provider下面的新注册的服务
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, DubboNodes[PROVIDER])
	c.Lock()
	err = c.client.Create(dubboPath)
	c.Unlock()
	if err != nil {
		log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}

	params = url.Values{}
	params.Add("interface", conf.Service)
	params.Add("application", c.ApplicationConfig.Name)
	revision = c.ApplicationConfig.Version
	if revision == "" {
		revision = "0.1.0"
	}
	params.Add("revision", revision)
	if conf.Group != "" {
		params.Add("group", conf.Group)
	}
	params.Add("category", (DubboType(CONSUMER)).String())
	params.Add("dubbo", "dubbo-consumer-golang-"+version.Version)
	params.Add("org", c.Organization)
	params.Add("module", c.Module)
	params.Add("owner", c.Owner)
	params.Add("side", (DubboType(CONSUMER)).Role())
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("timeout", fmt.Sprintf("%v", c.Timeout))
	// params.Add("timestamp", time.Now().Format("20060102150405"))
	params.Add("timestamp", fmt.Sprintf("%d", c.birth))
	if conf.Version != "" {
		params.Add("version", conf.Version)
	}
	// log.Debug("consumer zk url params:%#v", params)
	rawURL = fmt.Sprintf("%s://%s/%s?%s", conf.Protocol, localIP, conf.Service+conf.Version, params.Encode())
	encodedURL = url.QueryEscape(rawURL)
	// log.Debug("url.QueryEscape(consumer url:%s) = %s", rawURL, encodedURL)

	// 把自己注册service consumers里面
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, (DubboType(CONSUMER)).String())
	log.Debug("consumer path:%s, url:%s", dubboPath, rawURL)
	err = c.registerTempZookeeperNode(dubboPath, encodedURL)
	if err != nil {
		return jerrors.Trace(err)
	}

	return nil
}

func (c *consumerZookeeperRegistry) handleZkRestart() {
	var (
		err       error
		flag      bool
		failTimes int
		confIf    registry.ServiceConfigIf
		services  []registry.ServiceConfigIf
	)

	defer c.wg.Done()
LOOP:
	for {
		select {
		case <-c.done:
			log.Warn("(consumerZookeeperRegistry)reconnectZkRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-c.client.done():
			c.Lock()
			c.client.Close()
			c.client = nil
			c.Unlock()

			// 接zk，直至成功
			failTimes = 0
			for {
				select {
				case <-c.done:
					log.Warn("(consumerZookeeperRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-time.After(common.TimeSecondDuration(failTimes * registry.REGISTRY_CONN_DELAY)): // 防止疯狂重连zk
				}
				err = c.validateZookeeperClient()
				log.Info("consumerZookeeperRegistry.validateZookeeperClient(zkAddrs{%s}) = error{%#v}",
					c.client.zkAddrs, jerrors.ErrorStack(err))
				if err == nil {
					// copy c.services
					c.Lock()
					for _, confIf = range c.services {
						services = append(services, confIf)
					}
					c.Unlock()

					flag = true
					for _, confIf = range services {
						err = c.register(confIf.(*registry.ServiceConfig))
						if err != nil {
							log.Error("in (consumerZookeeperRegistry)reRegister, (consumerZookeeperRegistry)register(conf{%#v}) = error{%#v}",
								confIf.(*registry.ServiceConfig), jerrors.ErrorStack(err))
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

func (c *consumerZookeeperRegistry) Watch() (registry.Watcher, error) {
	var (
		ok          bool
		err         error
		dubboPath   string
		client      *zookeeperClient
		iWatcher    registry.Watcher
		zkWatcher   *zookeeperWatcher
		serviceConf *registry.ServiceConfig
	)

	// new client & watcher
	client, err = newZookeeperClient(WatcherZkClient, c.Address, c.RegistryConfig.Timeout)
	if err != nil {
		log.Warn("newZookeeperClient(name:%s, zk addresss{%v}, timeout{%d}) = error{%v}",
			WatcherZkClient, c.Address, c.Timeout, jerrors.ErrorStack(err))
		return nil, jerrors.Trace(err)
	}
	iWatcher, err = newZookeeperWatcher(client)
	if err != nil {
		client.Close()
		log.Warn("newZookeeperWatcher() = error{%v}", jerrors.ErrorStack(err))
		return nil, jerrors.Trace(err)
	}
	zkWatcher = iWatcher.(*zookeeperWatcher)

	// watch
	c.Lock()
	for _, service := range c.services {
		// 监控相关服务的providers
		if serviceConf, ok = service.(*registry.ServiceConfig); ok {
			dubboPath = fmt.Sprintf("/dubbo/%s/providers", serviceConf.Service)
			log.Info("watch dubbo provider path{%s} and wait to get all provider zk nodes", dubboPath)
			// watchService过程中会产生event，如果zookeeperWatcher{events} channel被塞满，
			// 但是selector还没有准备好接收，下面这个函数就会阻塞，所以起动一个gr以防止阻塞for-loop
			go zkWatcher.watchService(dubboPath, *serviceConf)
		}
	}
	c.Unlock()

	return iWatcher, nil
}

// name: service@protocol
func (c *consumerZookeeperRegistry) GetServices(i registry.ServiceConfigIf) ([]*registry.ServiceURL, error) {
	var (
		ok            bool
		err           error
		dubboPath     string
		nodes         []string
		serviceURL    *registry.ServiceURL
		serviceConfIf registry.ServiceConfigIf
		sc            *registry.ServiceConfig
		serviceConf   *registry.ServiceConfig
	)

	sc, ok = i.(*registry.ServiceConfig)
	if !ok {
		return nil, jerrors.Errorf("@i:%#v is not of type registry.ServiceConfig type", i)
	}

	c.Lock()
	for k, v := range c.services {
		log.Debug("(consumerZookeeperRegistry)GetServices, service{%q}, serviceURL{%s}", k, v)
	}
	serviceConfIf, ok = c.services[sc.Key()]
	c.Unlock()
	if !ok {
		return nil, jerrors.Errorf("Service{%s} has not been registered", sc.Key())
	}
	serviceConf, ok = serviceConfIf.(*registry.ServiceConfig)
	if !ok {
		return nil, jerrors.Errorf("Service{%s}: failed to get serviceConfigIf type", sc.Key())
	}

	dubboPath = fmt.Sprintf("/dubbo/%s/providers", sc.Service)
	err = c.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	c.Lock()
	nodes, err = c.client.getChildren(dubboPath)
	c.Unlock()
	if err != nil {
		log.Warn("getChildren(dubboPath{%s}) = error{%v}", dubboPath, err)
		return nil, jerrors.Trace(err)
	}

	var serviceMap = make(map[string]*registry.ServiceURL)
	for _, n := range nodes {
		serviceURL, err = registry.NewServiceURL(n)
		if err != nil {
			log.Error("NewServiceURL({%s}) = error{%v}", n, err)
			continue
		}
		if !serviceConf.ServiceEqual(serviceURL) {
			log.Warn("serviceURL{%s} is not compatible with ServiceConfig{%#v}", serviceURL, serviceConf)
			continue
		}

		_, ok := serviceMap[serviceURL.Query.Get(serviceURL.Location)]
		if !ok {
			serviceMap[serviceURL.Location] = serviceURL
			continue
		}
	}

	var services []*registry.ServiceURL
	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (c *consumerZookeeperRegistry) String() string {
	return "dubbogo-consumer-zookeeper-registry"
}

// 删除zk上注册的registers
func (c *consumerZookeeperRegistry) closeRegisters() {
	c.Lock()
	log.Info("begin to close consumer zk client")
	// 先关闭旧client，以关闭tmp node
	c.client.Close()
	c.client = nil
	//for key = range c.services {
	//	// 	delete(c.services, key)
	//	log.Debug("delete register consumer zk path:%s", key)
	//}
	c.services = nil
	c.Unlock()
}

func (c *consumerZookeeperRegistry) Close() {
	close(c.done)
	c.wg.Wait()
	c.closeRegisters()
}
