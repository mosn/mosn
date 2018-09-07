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
	"os"
	"sync"
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

//////////////////////////////////////////////
// DubboType
//////////////////////////////////////////////

type DubboType int

const (
	CONSUMER = iota
	CONFIGURATOR
	ROUTER
	PROVIDER
)

var (
	DubboNodes       = [...]string{"consumers", "configurators", "routers", "providers"}
	DubboRole        = [...]string{"consumer", "", "", "provider"}
	RegistryZkClient = "zk registry"
	processID        = ""
	localIP          = ""
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = common.GetLocalIP(localIP)
}

func (t DubboType) String() string {
	return DubboNodes[t]
}

func (t DubboType) Role() string {
	return DubboRole[t]
}

//////////////////////////////////////////////
// zookeeperRegistry
//////////////////////////////////////////////

const (
	DEFAULT_REGISTRY_TIMEOUT = 1
)

// 从目前消费者的功能来看，它实现:
// 1 消费者在每个服务下的/dubbo/service/consumers下注册
// 2 消费者watch /dubbo/service/providers变动
// 3 zk连接创建的时候，监控连接的可用性
type zookeeperRegistry struct {
	common.ApplicationConfig
	registry.RegistryConfig                // ZooKeeperServers []string
	birth                   int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg                      sync.WaitGroup // wg+done for zk restart
	done                    chan struct{}
	sync.Mutex              // lock for client + services
	client                  *zookeeperClient
	services                map[string]registry.ServiceConfigIf // service name + protocol -> service config
}

func newZookeeperRegistry(opts registry.Options) (*zookeeperRegistry, error) {
	var (
		err error
		r   *zookeeperRegistry
	)

	r = &zookeeperRegistry{
		RegistryConfig:    opts.RegistryConfig,
		ApplicationConfig: opts.ApplicationConfig,
		birth:             time.Now().Unix(),
		done:              make(chan struct{}),
	}
	if r.Name == "" {
		r.Name = version.Name
	}
	if r.Version == "" {
		r.Version = version.Version
	}
	if r.RegistryConfig.Timeout == 0 {
		r.RegistryConfig.Timeout = DEFAULT_REGISTRY_TIMEOUT
	}
	err = r.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	r.services = make(map[string]registry.ServiceConfigIf)

	return r, nil
}

func (r *zookeeperRegistry) validateZookeeperClient() error {
	var (
		err error
	)

	err = nil
	r.Lock()
	defer r.Unlock()
	if r.client == nil {
		r.client, err = newZookeeperClient(RegistryZkClient, r.Address, r.RegistryConfig.Timeout)
		if err != nil {
			log.Warn("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%v}",
				RegistryZkClient, r.Address, r.Timeout, jerrors.ErrorStack(err))
		}
	}

	return jerrors.Annotatef(err, "newZookeeperClient(address:%+v)", r.Address)
}

func (r *zookeeperRegistry) Close() {
	r.client.Close()
}

func (r *zookeeperRegistry) registerZookeeperNode(root string, data []byte) error {
	var (
		err    error
		zkPath string
	)

	// 假设root是/dubbo/com.ofpay.demo.api.UserProvider/consumers/jsonrpc，则创建完成的时候zkPath
	// 是/dubbo/com.ofpay.demo.api.UserProvider/consumers/jsonrpc/0000000000之类的临时节点.
	// 这个节点在连接有效的时候回一直存在，直到退出的时候才会被删除。
	// 所以如果连接有效，欲删除/dubbo/com.ofpay.demo.api.UserProvider/consumers/jsonrpc的话，必须先把这个临时节点删除掉
	r.Lock()
	defer r.Unlock()
	err = r.client.Create(root)
	if err != nil {
		log.Error("zk.Create(root{%s}) = err{%v}", root, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "zkclient.Create(root:%s)", root)
	}
	zkPath, err = r.client.RegisterTempSeq(root, data)
	// 创建完临时节点，zkPath = /dubbo/com.ofpay.demo.api.UserProvider/consumers/jsonrpc/0000000000
	if err != nil {
		log.Error("createTempSeqNode(root{%s}) = error{%v}", root, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "createTempSeqNode(root{%s})", root)
	}
	// r.registers[root] = string(data) // root = /dubbo/com.ofpay.demo.api.UserProvider/consumers/jsonrpc
	log.Debug("create a zookeeper node:%s", zkPath)

	return nil
}

func (r *zookeeperRegistry) registerTempZookeeperNode(root string, node string) error {
	var (
		err    error
		zkPath string
	)

	r.Lock()
	defer r.Unlock()
	err = r.client.Create(root)
	if err != nil {
		log.Error("zk.Create(root{%s}) = err{%v}", root, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "zk.Create(root{%s})", root)
		return err
	}
	zkPath, err = r.client.RegisterTemp(root, node)
	if err != nil {
		log.Error("RegisterTempNode(root{%s}, node{%s}) = error{%v}", root, node, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "RegisterTempNode(root{%s}, node{%s})", root, node)
	}
	// r.registers[zkPath] = ""
	log.Debug("create a zookeeper node:%s", zkPath)

	return nil
}

func (r *zookeeperRegistry) String() string {
	return "zookeeper-registry"
}
