package router

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"sync"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
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
