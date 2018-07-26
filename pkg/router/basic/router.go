/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package basic

import (
	"errors"
	"sync"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/router"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	//router.RegisterRouterConfigFactory(protocol.SofaRpc, NewRouters)
	//router.RegisterRouterConfigFactory(protocol.Http2, NewRouters)
	//router.RegisterRouterConfigFactory(protocol.Http1, NewRouters)
	//router.RegisterRouterConfigFactory(protocol.Xprotocol, NewRouters)
}

// types.Routers
type Routers struct {
	rMutex  *sync.RWMutex
	routers []router.RouteBase
}

//Routing, use static router first， then use dynamic router if support
func (rc *Routers) Route(headers map[string]string, randomValue uint64) types.Route {
	//use static router first, then use dynamic router
	for _, r := range rc.routers {
		if rule := r.Match(headers, randomValue); rule != nil {
			return rule
		}
	}

	return nil
}

func (rc *Routers) AddRouter(routerName string) {

	rc.rMutex.Lock()
	defer rc.rMutex.Unlock()

	for _, r := range rc.routers {
		if r.GetRouterName() == routerName {
			log.DefaultLogger.Debugf("[Basic Router]router already exist %s", routerName)

			return
		}
	}

	// new dynamic router
	// note: as cluster's name is ended with "@DEFAULT" @boqin ...check
	br := &basicRouter{
		name:    routerName,
		service: routerName,
		cluster: routerName,
	}

	if len(rc.routers) > 0 {
		br.globalTimeout = rc.routers[0].GlobalTimeout()
		br.policy = rc.routers[0].Policy().(*routerPolicy)
	} else {
		br.globalTimeout = types.GlobalTimeout
		br.policy = &routerPolicy{
			retryOn:      false,
			retryTimeout: 0,
			numRetries:   0,
		}
	}

	rc.routers = append(rc.routers, br)
	log.DefaultLogger.Debugf("[Basic Router]add routes,router name is %s, router %+v", br.name, br)
}

func (rc *Routers) DelRouter(routerName string) {
	rc.rMutex.Lock()
	defer rc.rMutex.Unlock()

	for i, r := range rc.routers {
		if r.GetRouterName() == routerName {
			//return
			rc.routers = append(rc.routers[:i], rc.routers[i+1:]...)
			log.DefaultLogger.Debugf("[Basic Router]delete routes,router name %s, routers is", routerName, rc)

			return
		}
	}
}

// types.Route
// types.RouteRule
// router.matchable
type basicRouter struct {
	RouteRuleImplAdaptor
	name          string
	service       string
	cluster       string
	globalTimeout time.Duration
	policy        *routerPolicy
}

func NewRouters(config interface{}) (types.Routers, error) {
	if config, ok := config.(*v2.Proxy); ok {
		routers := make([]router.RouteBase, 0)

		for _, r := range config.BasicRoutes {
			router := &basicRouter{
				name:          r.Name,
				service:       r.Service,
				cluster:       r.Cluster,
				globalTimeout: r.GlobalTimeout,
			}

			if r.RetryPolicy != nil {
				router.policy = &routerPolicy{
					retryOn:      r.RetryPolicy.RetryOn,
					retryTimeout: r.RetryPolicy.RetryTimeout,
					numRetries:   r.RetryPolicy.NumRetries,
				}
			} else {
				// default
				router.policy = &routerPolicy{
					retryOn:      false,
					retryTimeout: 0,
					numRetries:   0,
				}
			}

			routers = append(routers, router)
		}

		rc := &Routers{
			//new(sync.RWMutex),
			routers: routers,
		}

		//router.RoutersManager.AddRoutersSet(rc)
		return rc, nil

	}

	return nil, errors.New("invalid config struct")
}

func (srr *basicRouter) Match(headers map[string]string, randomValue uint64) types.Route {
	if headers == nil {
		return nil
	}

	var ok bool
	var service string

	if service, ok = headers["Service"]; !ok {
		if service, ok = headers["service"]; !ok {
			return nil
		}
	}

	if srr.service == service {
		return srr
	}

	return nil
}

func (srr *basicRouter) RedirectRule() types.RedirectRule {
	return nil
}

func (srr *basicRouter) RouteRule() types.RouteRule {
	return srr
}

func (srr *basicRouter) TraceDecorator() types.TraceDecorator {
	return nil
}

func (srr *basicRouter) ClusterName() string {
	return srr.cluster
}

func (srr *basicRouter) GlobalTimeout() time.Duration {
	return srr.globalTimeout
}

func (srr *basicRouter) Policy() types.Policy {
	return srr.policy
}

func (srr *basicRouter) GetRouterName() string {
	return srr.name
}

type routerPolicy struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
}

func (p *routerPolicy) RetryOn() bool {
	return p.retryOn
}

func (p *routerPolicy) TryTimeout() time.Duration {
	return p.retryTimeout
}

func (p *routerPolicy) NumRetries() uint32 {
	return p.numRetries
}

func (p *routerPolicy) RetryPolicy() types.RetryPolicy {
	return p
}

func (p *routerPolicy) ShadowPolicy() types.ShadowPolicy {
	return nil
}

func (p *routerPolicy) CorsPolicy() types.CorsPolicy {
	return nil
}

func (p *routerPolicy) LoadBalancerPolicy() types.LoadBalancerPolicy {
	return nil
}
