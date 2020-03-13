package store

import (
	"sync/atomic"
	"time"

	v2 "sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/config"
	"sofastack.io/sofa-mosn/pkg/featuregate"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/utils"
)

var fDump uint32

const ConfigAutoWrite featuregate.Feature = "auto_config"

func init() {
	feature := map[featuregate.Feature]featuregate.FeatureSpec{
		ConfigAutoWrite: {Default: false, PreRelease: featuregate.Alpha},
	}
	if err := featuregate.DefaultMutableFeatureGate.Add(feature); err != nil {
		panic(err)
	}
}

func tryDump() {
	dumpConfig()
}

func dumpConfig() {
	if !featuregate.DefaultMutableFeatureGate.Enabled(ConfigAutoWrite) {
		return
	}
	atomic.CompareAndSwapUint32(&fDump, 0, 1)
}

func doDumpConfig() {
	if atomic.CompareAndSwapUint32(&fDump, 1, 0) {
		mutex.Lock()
		defer mutex.Unlock()
		var listeners []v2.Listener
		var routers []*v2.RouterConfiguration
		var clusters []v2.Cluster
		log.DefaultLogger.Debugf("try to dump full config")
		// get listeners
		for _, l := range conf.Listener {
			listeners = append(listeners, l)
		}
		// get clusters
		for _, c := range conf.Cluster {
			clusters = append(clusters, c)
		}
		// get routers, should set the original path
		for name := range conf.Routers {
			routerPath := conf.routerConfigPath[name]
			r := conf.Routers[name] // copy the value
			r.RouterConfigPath = routerPath
			routers = append(routers, &r)
		}
		config.UpdateFullConfig(listeners, routers, clusters)
	}
}

func StartFeature() {
	if !featuregate.DefaultMutableFeatureGate.Enabled(ConfigAutoWrite) {
		return
	}
	utils.GoWithRecover(func() {
		log.DefaultLogger.Infof("auto write config when updated")
		for {
			doDumpConfig()
			time.Sleep(3 * time.Second)
		}
	}, nil)
}
