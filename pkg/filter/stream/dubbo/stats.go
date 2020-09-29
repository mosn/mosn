package dubbo

import (
	"sync"

	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/types"
)

var (
	ServiceInfo  = "request_total"
	ResponseSucc = "response_succ_total"
	ResponseFail = "response_fail_total"
	RequestTime  = "request_time"

	listenerKey = "listener"
	serviceKey  = "service"
	methodKey   = "method"
	subsetKey   = "subset"

	podSubsetKey = ""
	metricPre    = "mosn"
)

var (
	l            sync.RWMutex
	statsFactory = make(map[string]*Stats)
)

type Stats struct {
	RequestServiceInfo gometrics.Counter
	ResponseSucc       gometrics.Counter
	ResponseFail       gometrics.Counter
}

func getStats(listener, service, method string) *Stats {
	key := service + "-" + method

	l.RLock()
	s, ok := statsFactory[key]
	l.RUnlock()
	if ok {
		return s
	}

	l.Lock()
	defer l.Unlock()
	if s, ok = statsFactory[key]; ok {
		return s
	}

	lables := map[string]string{
		listenerKey: listener,
		serviceKey:  service,
		methodKey:   method,
	}
	if podSubsetKey != "" {
		pl := types.GetPodLabels()
		if pl[podSubsetKey] != "" {
			lables[subsetKey] = pl[podSubsetKey]
		}
	}

	mts, err := metrics.NewMetrics(metricPre, lables)
	if err != nil {
		log.DefaultLogger.Errorf("create metrics fail: labels:%v, err: %v", lables, err)
		statsFactory[key] = nil
		return nil
	}

	s = &Stats{
		RequestServiceInfo: mts.Counter(ServiceInfo),
		ResponseSucc:       mts.Counter(ResponseSucc),
		ResponseFail:       mts.Counter(ResponseFail),
	}
	statsFactory[key] = s
	return s
}
