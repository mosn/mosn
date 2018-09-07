// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxzookeeper provides a zookeeper registry
package gxzookeeper

import (
	"strings"
	"sync"
	//"io/ioutil"

	log "github.com/AlexStocks/log4go"

	"github.com/AlexStocks/goext/database/registry"
	"github.com/AlexStocks/goext/database/zookeeper"
	jerrors "github.com/juju/errors"
	"github.com/samuel/go-zookeeper/zk"
)

//////////////////////////////////////////////
// Registry
//////////////////////////////////////////////

type Registry struct {
	client          *gxzookeeper.Client
	options         gxregistry.Options
	sync.Mutex      // lock for client + register
	done            chan struct{}
	wg              sync.WaitGroup
	eventRegistry   map[string][]*chan struct{}
	serviceRegistry map[gxregistry.ServiceAttr]gxregistry.Service
}

func NewRegistry(opts ...gxregistry.Option) (gxregistry.Registry, error) {
	var (
		err     error
		r       *Registry
		conn    *zk.Conn
		event   <-chan zk.Event
		options gxregistry.Options
	)

	for _, o := range opts {
		o(&options)
	}
	if options.Addrs == nil {
		return nil, jerrors.Errorf("@options.Addrs is nil")
	}
	if options.Timeout == 0 {
		options.Timeout = gxregistry.DefaultTimeout
	}
	if options.Root == "" {
		options.Root = gxregistry.DefaultServiceRoot
	}
	// connect to zookeeper
	//zk.DefaultLogger = golog.New(ioutil.Discard, "[goext] ", golog.LstdFlags)
	conn, event, err = zk.Connect(options.Addrs, options.Timeout)
	if err != nil {
		return nil, jerrors.Annotatef(err, "zk.Connect(zk addr:%#v, timeout:%d)",
			options.Addrs, options.Timeout)
	}
	r = &Registry{
		options:         options,
		client:          gxzookeeper.NewClient(conn),
		done:            make(chan struct{}),
		eventRegistry:   make(map[string][]*chan struct{}),
		serviceRegistry: make(map[gxregistry.ServiceAttr]gxregistry.Service),
	}
	go r.handleZkEvent(event)

	return r, nil
}

func (r *Registry) registerEvent(path string, event *chan struct{}) {
	if path == "" || event == nil {
		return
	}

	r.Lock()
	a := r.eventRegistry[path]
	a = append(a, event)
	r.eventRegistry[path] = a
	r.Unlock()
	log.Debug("zkClient register event{path:%s, ptr:%p}", path, event)
}

func (r *Registry) unregisterEvent(path string, event *chan struct{}) {
	if path == "" {
		return
	}

	r.Lock()
	for {
		a, ok := r.eventRegistry[path]
		if !ok {
			break
		}
		for i, e := range a {
			if e == event {
				arr := a
				a = append(arr[:i], arr[i+1:]...)
				log.Debug("zkClient unregister event{path:%s, event:%p}", path, event)
			}
		}
		log.Debug("after zkClient unregister event{path:%s, event:%p}, array length %d", path, event, len(a))
		if len(a) == 0 {
			delete(r.eventRegistry, path)
		} else {
			r.eventRegistry[path] = a
		}
		break
	}
	r.Unlock()
}

func (r *Registry) handleZkRestart() {
	var (
		err      error
		services []gxregistry.Service
	)

	// copy c.services
	services = make([]gxregistry.Service, 0, len(r.serviceRegistry))
	r.Lock()
	for _, s := range r.serviceRegistry {
		services = append(services, s)
	}
	r.Unlock()

	for _, s := range services {
		err = r.register(s)
		if err != nil {
			log.Error("(ZookeeperRegistry)register(service:%s) = error:%s", s, jerrors.ErrorStack(err))
		}
	}
}

func (r *Registry) handleZkEvent(session <-chan zk.Event) {
	var (
		state int
		event zk.Event
	)

	r.wg.Add(1)
	defer func() {
		r.wg.Done()
		log.Info("zk{addr:%#v, path:%v} connection goroutine game over.", r.options.Addrs, r.options.Root)
	}()

LOOP:
	for {
		select {
		case <-r.done:
			break LOOP

		case event = <-session:
			log.Warn("client get a zookeeper event{type:%s, server:%s, path:%s, state:%d-%s, err:%#v}",
				event.Type, event.Server, event.Path, event.State, r.client.StateToString(event.State), event.Err)
			switch (int)(event.State) {
			case (int)(zk.StateDisconnected):
				log.Warn("zk{addr:%#v, path:%v} state is StateDisconnected.", r.options.Addrs, r.options.Root)

			case (int)(zk.EventNodeDataChanged), (int)(zk.EventNodeChildrenChanged):
				log.Info("zkClient get zk node changed event{path:%s}", event.Path)
				r.Lock()
				for p, a := range r.eventRegistry {
					if strings.HasPrefix(p, event.Path) {
						log.Info("send event{zk.EventNodeDataChange, zk.Path:%s} to path{%s} related watcher", event.Path, p)
						for _, e := range a {
							*e <- struct{}{}
						}
					}
				}
				r.Unlock()

			case (int)(zk.StateConnecting), (int)(zk.StateConnected), (int)(zk.StateHasSession):
				if state != (int)(zk.StateConnecting) || state != (int)(zk.StateDisconnected) {
					continue
				}
				if a, ok := r.eventRegistry[event.Path]; ok && 0 < len(a) {
					for _, e := range a {
						*e <- struct{}{}
					}
				}
				if state == (int)(zk.StateDisconnected) && (int)(event.State) == (int)(zk.StateConnected) {
					log.Info("start to handle zookeeper restart event.")
					r.handleZkRestart()
				}
			}
			state = (int)(event.State)
		}
	}
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

	// serviceRegistry every node
	var zkPath string
	for i, node := range s.Nodes {
		service.Nodes = []*gxregistry.Node{node}
		data, err := gxregistry.EncodeService(&service)
		if err != nil {
			service.Nodes = s.Nodes[:i]
			r.unregister(service)
			return jerrors.Annotatef(err, "gxregistry.EncodeService(service:%+v) = error:%s", service, err)
		}

		zkPath = service.Path(r.options.Root)
		err = r.client.CreateZkPath(zkPath)
		if err != nil {
			log.Error("zkClient.CreateZkPath(root{%s})", zkPath, err)
			return jerrors.Trace(err)
		}

		zkPath = service.NodePath(r.options.Root, *node)
		_, err = r.client.RegisterTemp(zkPath, []byte(data))
		if err != nil {
			return jerrors.Annotatef(err, "gxregister.RegisterTemp(path:%s)", zkPath)
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

	var err error
	for _, node := range s.Nodes {
		err = r.client.DeleteZkPath(s.NodePath(r.options.Root, *node))
		if err != nil {
			return jerrors.Trace(err)
		}
	}

	return nil
}

func (r *Registry) Deregister(s gxregistry.Service) error {
	r.deleteService(s)
	return jerrors.Trace(r.unregister(s))
}

func (r *Registry) GetServices(attr gxregistry.ServiceAttr) ([]gxregistry.Service, error) {
	svc := gxregistry.Service{Attr: &attr}
	path := svc.Path(r.options.Root)
	children, err := r.client.GetChildren(path)
	if err != nil {
		return nil, jerrors.Annotatef(err, "zkClient.GetChildren(path:%s)", path)
	}
	if len(children) == 0 {
		return nil, gxregistry.ErrorRegistryNotFound
	}

	serviceArray := []gxregistry.Service{}
	var node gxregistry.Node
	for _, name := range children {
		node.ID = name
		zkPath := svc.NodePath(r.options.Root, node)

		childData, err := r.client.Get(zkPath)
		if err != nil {
			log.Warn("gxzookeeper.Get(name:%s) = error:%s", zkPath, jerrors.ErrorStack(err))
			continue
		}

		sn, err := gxregistry.DecodeService(childData)
		if err != nil {
			log.Warn("gxregistry.DecodeService(data:%#v) = error:%s", childData, jerrors.ErrorStack(err))
			continue
		}
		if attr.Filter(*sn.Attr) {
			for _, node := range sn.Nodes {
				var service gxregistry.Service
				service.Attr = sn.Attr
				service.Nodes = append(service.Nodes, node)
				serviceArray = append(serviceArray, service)
			}
		}
	}

	return serviceArray, nil
}

func (r *Registry) Watch(opts ...gxregistry.WatchOption) (gxregistry.Watcher, error) {
	w, err := NewWatcher(r, opts...)

	return w, jerrors.Trace(err)
}

func (r *Registry) String() string {
	return "zookeeper registry"
}

// check whether the session has been closed.
func (r *Registry) Done() <-chan struct{} {
	return r.done
}

func (r *Registry) Close() error {
	r.Lock()
	defer r.Unlock()
	if r.client != nil {
		close(r.done)
		r.client.ZkConn().Close()
		r.wg.Wait()
		r.client = nil
	}

	return nil
}
