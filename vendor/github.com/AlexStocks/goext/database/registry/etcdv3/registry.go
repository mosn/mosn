// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxetcd provides an etcd version 3 gxregistry
package gxetcd

import (
	"context"
	"strings"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	etcdv3 "github.com/coreos/etcd/clientv3"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/goext/database/etcd"
	"github.com/AlexStocks/goext/database/registry"
	"github.com/AlexStocks/goext/runtime"
)

type Registry struct {
	etcdClient *etcdv3.Client
	client     *gxetcd.Client
	options    gxregistry.Options

	done chan struct{}
	wg   sync.WaitGroup

	sync.Mutex
	sync.Once
	serviceRegistry map[gxregistry.ServiceAttr]gxregistry.Service
}

func NewRegistry(opts ...gxregistry.Option) (gxregistry.Registry, error) {
	config := etcdv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	}

	var options gxregistry.Options
	for _, o := range opts {
		o(&options)
	}
	if options.Timeout == 0 {
		options.Timeout = gxregistry.DefaultTimeout
	}

	var addrs []string
	for _, addr := range options.Addrs {
		if len(addr) == 0 {
			continue
		}
		addrs = append(addrs, addr)
	}

	// if we got addrs then we'll update
	if len(addrs) > 0 {
		config.Endpoints = addrs
	}
	config.DialTimeout = options.Timeout
	config.DialKeepAliveTimeout = options.Timeout

	client, err := etcdv3.New(config)
	if err != nil {
		return nil, jerrors.Errorf("etcdv3.New(config:%+v) = error:%s", config, err)
	}
	gxClient, err := gxetcd.NewClient(client, gxetcd.WithTTL(options.Timeout))
	if err != nil {
		return nil, jerrors.Errorf("gxetcd.NewClient() = error:%s", err)
	}

	if options.Root == "" {
		options.Root = gxregistry.DefaultServiceRoot
	}

	r := &Registry{
		etcdClient:      client,
		client:          gxClient,
		options:         options,
		done:            make(chan struct{}),
		serviceRegistry: make(map[gxregistry.ServiceAttr]gxregistry.Service),
	}

	err = r.handleEtcdRestart()
	if err != nil {
		return nil, jerrors.Annotate(err, "registry.handleEtcdRestart()")
	}

	return r, nil
}

func (r *Registry) handleEtcdRestart() error {
	keepAlive, err := r.client.KeepAlive()
	if err != nil {
		return jerrors.Trace(err)
	}

	r.wg.Add(1)
	go func() {
		var (
			failTime     int
			registerFlag bool
		)
		defer r.wg.Done()

	LOOP:
		for {
			select {
			case <-r.done:
				log.Warn("Registry.done closed. Registry.handleEtcdRestart goroutine %d exit now ...", gxruntime.GoID())
				break LOOP
			case msg, ok := <-keepAlive:
				// eat messages until keep alive channel closes
				if !ok {
					registerFlag = true
					log.Warn("etcd keep alive channel closed")
					keepAlive, err = r.client.KeepAlive()
					if err != nil {
						log.Warn("gxetcd.KeepAlive() = error:%+v", err)
					}
					failTime <<= 1
					if failTime == 0 {
						failTime = 1e8
					} else if gxregistry.MaxFailTime < failTime {
						failTime = gxregistry.MaxFailTime
					}
					time.Sleep(time.Duration(failTime)) // to avoid connecting the registry tool frequently
				} else {
					failTime = 0
					// the etcd has restarted. now we need to register all services
					if registerFlag {
						services := []gxregistry.Service{}
						r.Lock()
						for _, s := range r.serviceRegistry {
							services = append(services, s)
						}
						r.Unlock()
						failure := false
						for idx := range services {
							if err = r.register(services[idx]); err != nil {
								failure = true
								log.Warn("Registry.register(service:{%#v}) = err:%+v", services[idx], err)
								break
							}
						}
						registerFlag = failure // when fails to register all services,  just register again.
					}
					log.Debug("Recv msg from etcd KeepAlive: %s\n", msg.String())
				}
			}
		}
	}()

	return nil
}

func (r *Registry) Options() gxregistry.Options {
	return r.options
}

// 如果s.nodes为空，则返回当前registry中的service
// 若非空，则检查其中的每个node是否存在
func (r *Registry) exist(s gxregistry.Service) (gxregistry.Service, bool) {
	// get existing hash
	r.Lock()
	defer r.Unlock()
	v, ok := r.serviceRegistry[*(s.Attr)]
	if len(s.Nodes) == 0 {
		return v, ok
	}

	for i := range s.Nodes {
		flag := false
		for j := range v.Nodes {
			if s.Nodes[i].Equal(v.Nodes[j]) {
				flag = true
				continue
			}
		}
		if !flag {
			// log.Error("s.node:%s, v.nodes:%s", gxlog.PrettyString(s.Nodes[i]), gxlog.PrettyString(v.Nodes))
			return v, false
		}
	}

	return v, true
}

func (r *Registry) addService(s gxregistry.Service) {
	if len(s.Nodes) == 0 {
		return
	}

	// get existing hash
	r.Lock()
	defer r.Unlock()
	v, ok := r.serviceRegistry[*s.Attr]
	if !ok {
		r.serviceRegistry[*s.Attr] = *(s.Copy())
		return
	}

	for i := range s.Nodes {
		flag := false
		for j := range v.Nodes {
			if s.Nodes[i].Equal(v.Nodes[j]) {
				flag = true
			}
		}
		if !flag {
			v.Nodes = append(v.Nodes, s.Nodes[i].Copy())
		}
	}
	r.serviceRegistry[*s.Attr] = v

	return
}

func (r *Registry) deleteService(s gxregistry.Service) {
	if len(s.Nodes) == 0 {
		return
	}

	// get existing hash
	r.Lock()
	defer r.Unlock()
	v, ok := r.serviceRegistry[*s.Attr]
	if !ok {
		return
	}

	for i := range s.Nodes {
		for j := range v.Nodes {
			if s.Nodes[i].Equal(v.Nodes[j]) {
				v.Nodes = append(v.Nodes[:j], v.Nodes[j+1:]...)
				break
			}
		}
	}
	r.serviceRegistry[*s.Attr] = v

	return
}

func (r *Registry) register(s gxregistry.Service) error {
	service := gxregistry.Service{Metadata: s.Metadata}
	service.Attr = s.Attr

	ctx, cancel := context.WithTimeout(context.Background(), r.options.Timeout)
	defer cancel()

	// serviceRegistry every node
	for i, node := range s.Nodes {
		service.Nodes = []*gxregistry.Node{node}
		data, err := gxregistry.EncodeService(&service)
		if err != nil {
			service.Nodes = s.Nodes[:i]
			r.unregister(service)
			return jerrors.Annotatef(err, "gxregistry.EncodeService(service:%+v) = error:%s", service, err)
		}
		_, err = r.client.EtcdClient().Put(
			ctx,
			service.NodePath(r.options.Root, *node),
			data,
			etcdv3.WithLease(r.client.Lease()),
		)
		if err != nil {
			service.Nodes = s.Nodes[:i]
			r.unregister(service)
			return jerrors.Annotatef(err, "etcdv3.Client.Put(path, data)",
				service.NodePath(r.options.Root, *node), data)
		}
	}

	return nil
}

func (r *Registry) Register(s gxregistry.Service) error {
	if len(s.Nodes) == 0 {
		return jerrors.Errorf("Require at least one node")
	}

	if _, exist := r.exist(s); exist {
		return gxregistry.ErrorAlreadyRegister
	}

	err := r.register(s)
	if err != nil {
		return jerrors.Annotate(err, "Registry.register")
	}

	// save the service
	r.addService(s)

	return nil
}

func (r *Registry) unregister(s gxregistry.Service) error {
	if len(s.Nodes) == 0 {
		return jerrors.Errorf("Require at least one node")
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.options.Timeout)
	defer cancel()

	for _, node := range s.Nodes {
		_, err := r.client.EtcdClient().Delete(ctx, s.NodePath(r.options.Root, *node), etcdv3.WithIgnoreLease())
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) Deregister(s gxregistry.Service) error {
	r.deleteService(s)
	return jerrors.Trace(r.unregister(s))
}

func (r *Registry) GetServices(attr gxregistry.ServiceAttr) ([]gxregistry.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.options.Timeout)
	defer cancel()

	// svc := gxregistry.Service{Attr: &attr}
	path := r.options.Root
	if !strings.HasPrefix(path, "/") {
		path += "/"
	}
	rsp, err := r.client.EtcdClient().Get(
		ctx,
		path,
		etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortDescend),
	)
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return nil, gxregistry.ErrorRegistryNotFound
	}

	services := make([]gxregistry.Service, 0, 32)
	for _, n := range rsp.Kvs {
		if sn, err := gxregistry.DecodeService(n.Value); err == nil && sn != nil {
			if attr.Filter(*sn.Attr) {
				for _, node := range sn.Nodes {
					var service gxregistry.Service
					service.Attr = sn.Attr // bug fix: @attr 仅仅用于过滤，其属性值比较少，etcd 返回的service.ServiceAttr 值比 @attr 精确
					service.Nodes = append(service.Nodes, node)
					services = append(services, service)
				}
			}
		}
	}

	return services, nil
}

func (r *Registry) Watch(opts ...gxregistry.WatchOption) (gxregistry.Watcher, error) {
	return NewWatcher(r.client, opts...)
}

func (r *Registry) String() string {
	return "Etcdv3 Registry"
}

func (r *Registry) Close() error {
	var err error
	r.Lock()
	if r.etcdClient != nil {
		close(r.done)
		err = r.client.Close()
		if err != nil {
			err = jerrors.Annotate(err, "gxetcd.Client.Close()")
		}
		r.etcdClient.Close()
		r.wg.Wait()
		r.etcdClient = nil
	}
	r.Unlock()

	return err
}
