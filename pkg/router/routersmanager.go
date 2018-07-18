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

package router

import (
	"sync"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

var RoutersManager = routersManager{
	rMutex: new(sync.RWMutex),
}

type routersManager struct {
	routersSet  []types.Routers
	rMutex      *sync.RWMutex
	routerNames []string
}

func (rm *routersManager) AddRoutersSet(routers types.Routers) {
	rm.rMutex.RLock()
	defer rm.rMutex.RUnlock()
	//log.StartLogger.Debugf("[RouterManager RoutersSet]Add RoutersSet Called, routers is %+v",routers)

	for _, name := range rm.routerNames {
		routers.AddRouter(name)
	}

	rm.routersSet = append(rm.routersSet, routers)
}

func (rm *routersManager) RemoveRouterInRouters(routerNames []string) {
	log.DefaultLogger.Debugf("[RouterManager]RemoveRouterInRouters Called, routerNames is", routerNames)

	rm.DeleteRouterNameInList(routerNames)
	for _, rn := range routerNames {
		for _, r := range rm.routersSet {
			r.DelRouter(rn)
		}
	}
}

func (rm *routersManager) AddRouterInRouters(routerNames []string) {

	log.DefaultLogger.Debugf("[RouterManager]AddRouterInRouters Called, routerNames is", routerNames)
	rm.AddRouterNameInList(routerNames)

	for _, rn := range routerNames {
		for _, r := range rm.routersSet {
			//	log.StartLogger.Debugf("[RouterManager]Add Routers to Basic Router %+v", r)
			r.AddRouter(rn)
		}
	}
}

func (rm *routersManager) AddRouterNameInList(routerNames []string) {
	rm.rMutex.Lock()
	defer rm.rMutex.Unlock()

	for _, name := range routerNames {
		in := false
		for _, n := range rm.routerNames {
			if name == n {
				in = true
				break
			}
		}
		if !in {
			rm.routerNames = append(rm.routerNames, name)
		}
	}
}

func (rm *routersManager) DeleteRouterNameInList(rNames []string) {
	rm.rMutex.Lock()
	defer rm.rMutex.Unlock()

	for _, name := range rNames {
		for i, n := range rm.routerNames {
			if name == n {
				rm.routerNames = append(rm.routerNames[:i], rm.routerNames[i+1:]...)
				break
			}
		}
	}
}
