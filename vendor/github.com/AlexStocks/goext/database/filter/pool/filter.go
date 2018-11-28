// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxpool provides a service pool filter
package gxpool

import (
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/filter"
	"github.com/AlexStocks/goext/database/registry"
	"github.com/AlexStocks/goext/time"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

var (
	defaultTTL = 10 * time.Minute
)

/////////////////////////////////////
// Filter
/////////////////////////////////////

type Filter struct {
	// registry & strategy
	opts gxfilter.Options
	ttl  time.Duration

	wg sync.WaitGroup
	sync.Mutex
	serviceMap map[string]*gxfilter.ServiceArray

	done chan struct{}
}

func (s *Filter) isClosed() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// copy is invoked by function get.
func (s *Filter) copy(current []*gxregistry.Service, by gxregistry.ServiceAttr) []*gxregistry.Service {
	var services []*gxregistry.Service

	for _, service := range current {
		service := service
		if by.Filter(*service.Attr) {
			s := *service
			s.Nodes = make([]*gxregistry.Node, 0, len(service.Nodes))
			for _, node := range service.Nodes {
				n := *node
				s.Nodes = append(s.Nodes, &n)
			}

			services = append(services, &s)
		}
	}

	return services
}

func (s *Filter) get(attr gxregistry.ServiceAttr) (*gxfilter.ServiceArray, error) {
	s.Lock()
	serviceString := attr.Service
	serviceArray, sok := s.serviceMap[serviceString]
	log.Debug("s.serviceMap[serviceString{%v}] = services{%}", serviceString, serviceArray)

	if sok && len(serviceArray.Arr) > 0 {
		ttl := time.Since(serviceArray.Active)
		if ttl < s.ttl {
			s.Unlock()
			return &gxfilter.ServiceArray{Arr: s.copy(serviceArray.Arr, attr), Active: serviceArray.Active}, nil
		}
		log.Warn("s.serviceMap[serviceString{%v}] = services{%s}, array ttl{%v} is less than services.ttl{%v}",
			serviceString, serviceArray, ttl, s.ttl)
	}
	s.Unlock()

	svcs, err := s.opts.Registry.GetServices(attr)
	s.Lock()
	defer s.Unlock()
	if err != nil {
		log.Error("registry.GetService(serviceString{%v}) = err:%+v}", serviceString, err)
		if sok && len(serviceArray.Arr) > 0 {
			log.Error("serviceString{%v} timeout. can not get new serviceString array, use old instead", serviceString)
			// all local services expired and can not get new services from registry, just use che cached instead
			return &gxfilter.ServiceArray{Arr: s.copy(serviceArray.Arr, attr), Active: serviceArray.Active}, nil
		}

		return nil, jerrors.Annotatef(err, "cacheSelect.get(ServiceAttr:%+v)", attr)
	}

	var arr []*gxregistry.Service
	for i, svc := range svcs {
		svc := svc
		arr = append(arr, &svc)
		log.Debug("i:%d, svc:%+v, service array:%+v", i, svc, arr)
	}

	filterServiceArray := s.copy(arr, attr)
	s.serviceMap[serviceString] = gxfilter.NewServiceArray(arr)

	return &gxfilter.ServiceArray{Arr: filterServiceArray, Active: s.serviceMap[serviceString].Active}, nil
}

func (s *Filter) update(res *gxregistry.EventResult) {
	if res == nil || res.Service == nil {
		return
	}
	var (
		ok           bool
		name         string
		serviceArray *gxfilter.ServiceArray
	)

	name = res.Service.Attr.Service
	s.Lock()
	serviceArray, ok = s.serviceMap[name]

	switch res.Action {
	case gxregistry.ServiceAdd, gxregistry.ServiceUpdate:
		if ok {
			serviceArray.Add(res.Service, s.ttl)
			log.Info("filter add serviceURL{%#v}", *res.Service)
		} else {
			s.serviceMap[name] = gxfilter.NewServiceArray([]*gxregistry.Service{res.Service})
		}
	case gxregistry.ServiceDel:
		if ok {
			serviceArray.Del(res.Service, s.ttl)
			if len(serviceArray.Arr) == 0 {
				delete(s.serviceMap, name)
				log.Warn("delete service %s from service map", name)
			}
		}
		log.Warn("filter delete serviceURL{%#v}", *res.Service)
	}
	s.Unlock()
}

func (s *Filter) run(attr gxregistry.ServiceAttr) {
	defer s.wg.Done()
	for {
		// quit asap
		if s.isClosed() {
			log.Warn("(Filter)run() isClosed now")
			return
		}

		w, err := s.opts.Registry.Watch(
			gxregistry.WithWatchRoot(s.opts.Registry.Options().Root),
			gxregistry.WithWatchFilter(attr),
		)
		log.Debug("services.Registry.Watch() = watch:%+v, error:%+v", w, jerrors.ErrorStack(err))
		if err != nil {
			if s.isClosed() {
				log.Warn("(Filter)run() isClosed now")
				return
			}
			log.Warn("Registry.Watch() = error:%+v", jerrors.ErrorStack(err))
			time.Sleep(gxtime.TimeSecondDuration(gxregistry.REGISTRY_CONN_DELAY))
			continue
		}

		// this function will block until got done signal
		err = s.watch(w)
		log.Debug("services.watch(w) = err{%#+v}", jerrors.ErrorStack(err))
		if err != nil {
			log.Warn("Filter.watch() = error{%v}", jerrors.ErrorStack(err))
			time.Sleep(gxtime.TimeSecondDuration(gxregistry.REGISTRY_CONN_DELAY))
			continue
		}
	}
}

func (s *Filter) watch(w gxregistry.Watcher) error {
	var (
		err  error
		res  *gxregistry.EventResult
		done chan struct{}
	)
	// manage this loop
	done = make(chan struct{})

	defer func() {
		close(done)
		w.Close() // 此处与 line241 冲突，可能导致两次 close watcher
	}()
	s.wg.Add(1)
	go func() {
		select {
		case <-s.done:
			w.Close()
		case <-done:
		}
		s.wg.Done()
	}()

	for {
		res, err = w.Notify()
		log.Debug("watch.Notify() = result{%s}, error{%#v}", res, err)
		if err != nil {
			return err
		}
		if res.Action == gxregistry.ServiceDel && !w.Valid() {
			// do not delete any provider when consumer failed to connect the registry.
			log.Warn("update @result{%s}. But its connection to registry is invalid", res)
			continue
		}

		s.update(res)
	}
}

func (s *Filter) Options() gxfilter.Options {
	return s.opts
}

func (s *Filter) GetService(service gxregistry.ServiceAttr) ([]*gxregistry.Service, error) {
	serviceArray, err := s.Filter(service)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return serviceArray.Arr, nil
}

func (s *Filter) Filter(service gxregistry.ServiceAttr) (*gxfilter.ServiceArray, error) {
	var (
		err      error
		svcArray *gxfilter.ServiceArray
	)

	svcArray, err = s.get(service)
	if err != nil {
		log.Error("services.get(service{%s}) = error{%s}", service, jerrors.ErrorStack(err))
		return nil, gxfilter.ErrNotFound
	}
	if len(svcArray.Arr) == 0 {
		return nil, gxfilter.ErrNoneAvailable
	}

	return svcArray, nil
}

func (s *Filter) CheckServiceAlive(attr gxregistry.ServiceAttr, svcArray *gxfilter.ServiceArray) bool {
	var (
		ok     bool
		arr    *gxfilter.ServiceArray
		t0, t1 time.Time
	)

	s.Lock()
	arr, ok = s.serviceMap[attr.Service]
	s.Unlock()

	if ok {
		t0 = arr.Active
		t1 = svcArray.Active
		if t0.Sub(t1) > 0 {
			// Active expire, the user should update its service array
			return false
		}
		return true
	}

	return false
}

func (s *Filter) Close() error {
	s.Lock()
	s.serviceMap = make(map[string]*gxfilter.ServiceArray)
	s.Unlock()

	select {
	case <-s.done:
		return nil
	default:
		close(s.done)
	}
	s.wg.Wait()
	return nil
}

func NewFilter(opts ...gxfilter.Option) (gxfilter.Filter, error) {
	sopts := gxfilter.Options{}

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		return nil, jerrors.Errorf("@opts.Registry is nil")
	}

	ttl := defaultTTL
	if sopts.Context != nil {
		if t, ok := sopts.Context.Get(GxfilterDefaultKey); ok {
			ttl = t.(time.Duration)
		}
	}

	s := &Filter{
		opts:       sopts,
		ttl:        ttl,
		serviceMap: make(map[string]*gxfilter.ServiceArray),
		done:       make(chan struct{}),
	}

	var serviceAttr gxregistry.ServiceAttr
	if sopts.Context != nil {
		if attr, ok := sopts.Context.Get(GxfilterServiceAttrKey); ok {
			serviceAttr = attr.(gxregistry.ServiceAttr)
		}
	}

	s.wg.Add(1)
	go s.run(serviceAttr)
	return s, nil
}
