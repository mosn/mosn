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

package cache

import (
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
	"github.com/AlexStocks/dubbogo/selector"
)

var (
	// selector每DefaultTTL分钟通过tick函数清空cache或者get函数去清空某个service的cache，
	// 以全量获取某个service的所有providers
	DefaultTTL = 10 * time.Minute
)

/*
	Cache selector is a selector which uses the registry.Watcher to Cache service entries.
	It defaults to a TTL for DefaultTTL and causes a cache miss on the next request.
*/
type cacheSelector struct {
	so  selector.Options // registry & strategy
	ttl time.Duration

	// registry cache
	wg sync.WaitGroup
	sync.Mutex
	cache map[string][]*registry.ServiceURL
	ttls  map[string]time.Time // 每个数组的创建时间

	// used to close or reload watcher
	exit chan bool
}

func (c *cacheSelector) quit() bool {
	select {
	case <-c.exit:
		return true
	default:
		return false
	}
}

// copy copies a service. Because we're caching handing back pointers would
// create a race condition, so we do this instead its fast enough
// 这个函数会被下面的get函数调用，get调用期间会启用lock
func (c *cacheSelector) copy(current []*registry.ServiceURL) []*registry.ServiceURL {
	var services []*registry.ServiceURL

	for _, service := range current {
		service := service
		services = append(services, service)
	}

	return services
}

func (c *cacheSelector) get(s registry.ServiceConfigIf) ([]*registry.ServiceURL, error) {
	c.Lock()
	defer c.Unlock()

	log.Debug("c.get(@s:%#v)", s)

	// check the cache first
	var serviceConf registry.ServiceConfig
	if scp, ok := s.(*registry.ServiceConfig); ok {
		serviceConf = *scp
	} else if sc, ok := s.(registry.ServiceConfig); ok {
		serviceConf = sc
	} else {
		return nil, jerrors.Errorf("illegal @s:%#v", s)
	}

	services, o := c.cache[serviceConf.Key()]
	ttl, k := c.ttls[serviceConf.Key()]
	// log.Debug("c.cache[service{%v}] = services{%v}", serviceConf.Service, services)
	log.Debug("c.cache[service{%#v}] = services{%v}", serviceConf, services)

	// got results, copy and return
	if o && len(services) > 0 {
		// only return if its less than the ttl
		// 拷贝services的内容，防止发生add/del event时影响results内容
		if k && time.Since(ttl) < c.ttl {
			return c.copy(services), nil
		}
		log.Warn("c.cache[serviceconf:%+v] = services:{%+v}, array ttl:{%+v} is less than cache.ttl:{%+v}",
			serviceConf, services, ttl, c.ttl)
		//serviceConf.Service, services, ttl, c.ttl)
	}

	// cache miss or ttl expired
	// now ask the registry
	ss, err := c.so.Registry.GetServices(s)
	if err != nil {
		log.Error("registry.GetServices(service{%#v}) = err{%v}", serviceConf, jerrors.ErrorStack(err))
		if o && len(services) > 0 {
			log.Error("service{%v} timeout. can not get new service array, use old instead", serviceConf.Service)
			return services, nil // 超时后，如果获取不到新的，就先暂用旧的
		}
		return nil, jerrors.Annotatef(err, "cacheSelect.get(serviceConfig:%+v)", serviceConf)
	}

	// we didn't have any results so cache
	c.cache[serviceConf.Key()] = c.copy(ss)
	c.ttls[serviceConf.Key()] = time.Now().Add(c.ttl)
	return ss, nil
}

// update函数调用set函数，update函数调用期间会启用lock
func (c *cacheSelector) set(service string, services []*registry.ServiceURL) {
	if 0 < len(services) {
		c.cache[service] = services
		c.ttls[service] = time.Now().Add(c.ttl)

		return
	}

	delete(c.cache, service)
	delete(c.ttls, service)
}

func filterServices(array *[]*registry.ServiceURL, i int) {
	if i < 0 {
		return
	}

	if len(*array) <= i {
		return
	}
	s := *array
	s = append(s[:i], s[i+1:]...)
	*array = s
}

func (c *cacheSelector) update(res *registry.Result) {
	if res == nil || res.Service == nil {
		return
	}
	var (
		ok       bool
		sname    string
		services []*registry.ServiceURL
	)

	log.Debug("update @registry result{%s}", res)
	sname = res.Service.ServiceConfig().Key()
	c.Lock()
	defer c.Unlock()
	services, ok = c.cache[sname]
	log.Debug("service name:%s, its current member lists:%+v", sname, services)
	if ok { // existing service found
		for i, s := range services {
			log.Debug("cache.services[%s][%d] = service{%s}", sname, i, s)
			if s.PrimitiveURL == res.Service.PrimitiveURL {
				filterServices(&(services), i)
			}
		}
	}

	switch res.Action {
	case registry.ServiceURLAdd, registry.ServiceURLUpdate:
		services = append(services, res.Service)
		log.Info("selector add serviceURL{%s}", *res.Service)
	case registry.ServiceURLDel:
		log.Error("selector delete serviceURL{%s}", *res.Service)
	}
	c.set(sname, services)
	//services, ok = c.cache[sname]
	log.Debug("after update, cache.services[%s] member list size{%d}", sname, len(c.cache[sname]))
	// if ok { // debug
	// 	for i, s := range services {
	// 		log.Debug("cache.services[%s][%d] = service{%#v}", sname, i, s)
	// 	}
	// }
}

// run starts the cache watcher loop
// it creates a new watcher if there's a problem
// reloads the watcher if Init(has been deleted) is called
// and returns when Close is called
func (c *cacheSelector) run() {
	defer c.wg.Done()

	// 除非收到quit信号，否则一直卡在watch上，watch内部也是一个loop
	for {
		// exit early if already dead
		if c.quit() {
			log.Warn("(cacheSelector)run() quit now")
			return
		}

		// create new watcher
		// 创建新watch，走到这一步要么第第一次for循环进来，要么是watch函数出错。watch出错说明registry.watch的zk client与zk连接出现了问题
		w, err := c.so.Registry.Watch()
		log.Debug("cache.Registry.Watch() = watch{%#v}, error{%#v}", w, jerrors.ErrorStack(err))
		if err != nil {
			if c.quit() {
				log.Warn("(cacheSelector)run() quit now")
				return
			}
			log.Warn("cacheSelector.Registry.Watch() = error{%v}", jerrors.ErrorStack(err))
			time.Sleep(common.TimeSecondDuration(registry.REGISTRY_CONN_DELAY))
			continue
		}

		// watch for events
		// 除非watch遇到错误，否则watch函数内部的for循环一直运行下午，run函数的for循环也会一直卡在watch函数上
		// watch一旦退出，就会执行registry.Watch().Stop, 相应的client就会无效
		err = c.watch(w)
		log.Debug("cache.watch(w) = err{%#+v}", jerrors.ErrorStack(err))
		if err != nil {
			log.Warn("cacheSelector.watch() = error{%v}", jerrors.ErrorStack(err))
			time.Sleep(common.TimeSecondDuration(registry.REGISTRY_CONN_DELAY))
			continue
		}
	}
}

// watch loops the next event and calls update
// it returns if there's an error
// 如果收到了退出信号或者Next调用返回错误，就调用watcher的stop函数
func (c *cacheSelector) watch(w registry.Watcher) error {
	var (
		err  error
		res  *registry.Result
		exit chan struct{}
	)
	// manage this loop
	exit = make(chan struct{})

	defer func() {
		close(exit)
		w.Stop()
	}()
	c.wg.Add(1)
	go func() {
		// wait for exit or reload signal
		// 上面的意思是这个goroutine会一直卡在这个select段上，直到收到exit或者reload signal
		select {
		case <-c.exit:
			w.Stop() // stop之后下面的Next函数就会返回error
		case <-exit:
		}
		c.wg.Done()
	}()

	for {
		res, err = w.Next()
		log.Debug("watch.Next() = result{%s}, error{%#v}", res, err)
		if err != nil {
			return jerrors.Trace(err)
		}
		if res.Action == registry.ServiceURLDel && !w.Valid() {
			// consumer与registry连接中断的情况下，为了服务的稳定不删除任何provider
			log.Warn("update @result{%s}. But its connection to registry is invalid", res)
			continue
		}

		c.update(res)
	}
}

func (c *cacheSelector) Options() selector.Options {
	return c.so
}

func (c *cacheSelector) Select(service registry.ServiceConfigIf) (selector.Next, error) {
	var (
		err      error
		services []*registry.ServiceURL
	)

	log.Debug("@service:%#v", service)

	// get the service
	// try the cache first
	// if that fails go directly to the registry
	services, err = c.get(service)
	//log.Debug("get(service{%+v} = serviceURL array{%#v})", service, services)
	if err != nil {
		log.Error("cache.get(service{%+v}) = error{%+v}", service, jerrors.ErrorStack(err))
		// return nil, err
		return nil, selector.ErrNotFound
	}
	// for i, s := range services {
	//	log.Debug("services[%d] = serviceURL{%#v}", i, s)
	// }
	// if there's nothing left, return
	if len(services) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	return selector.SelectorNext(c.so.Mode)(services), nil
}

// Close stops the watcher and destroys the cache
// Close函数清空service url cache，且发出exit signal以停止watch的运行
func (c *cacheSelector) Close() error {
	c.Lock()
	c.cache = make(map[string][]*registry.ServiceURL)
	c.Unlock()

	select {
	case <-c.exit:
		return nil
	default:
		close(c.exit)
	}
	c.wg.Wait()
	return nil
}

func (c *cacheSelector) String() string {
	return "cache-selector"
}

// selector主要有两个接口，对外接口Select用于获取地址，select调用get，get调用cp;
// 对内接口run调用watch,watch则调用update，update调用set，以用于接收add/del service url.
//
// 而Close则是发出stop信号，停掉watch，清算破产
//
// register自身主要向selector暴露了watch功能
func NewSelector(opts ...selector.Option) selector.Selector {
	sopts := selector.Options{
		Mode: selector.SM_Random,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		panic("@opts.Registry is nil")
	}

	ttl := DefaultTTL

	if sopts.Context != nil {
		if t, ok := sopts.Context.Value(common.DUBBOGO_CTX_KEY).(time.Duration); ok {
			ttl = t
		}
	}

	c := &cacheSelector{
		so:    sopts,
		ttl:   ttl,
		cache: make(map[string][]*registry.ServiceURL),
		ttls:  make(map[string]time.Time),
		exit:  make(chan bool),
	}

	c.wg.Add(1)
	go c.run()
	return c
}
