package grpc

import (
	"sync"

	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
)

var (
	ServiceReqNum = "request_total"
	ResponseSucc  = "response_succ_total"
	ResponseFail  = "response_fail_total"
	serviceKey    = "service"
	metricPre     = "grpc"
)

var (
	mux          sync.RWMutex
	statsFactory = make(map[string]*Stats)
)

type Stats struct {
	RequestServiceTootle gometrics.Counter
	ResponseSuccess      gometrics.Counter
	ResponseFail         gometrics.Counter
}

func getStats(service string) *Stats {
	key := service
	mux.RLock()
	s, ok := statsFactory[key]
	mux.RUnlock()
	if ok {
		return s
	}
	mux.Lock()
	defer mux.Unlock()
	if s, ok = statsFactory[key]; ok {
		return s
	}
	labels := map[string]string{
		serviceKey: key,
	}
	mts, err := metrics.NewMetrics(metricPre, labels)
	if err != nil {
		log.DefaultLogger.Errorf("create metrics fail: labels:%v, err: %v", labels, err)
		statsFactory[key] = nil
		return nil
	}

	s = &Stats{
		RequestServiceTootle: mts.Counter(ServiceReqNum),
		ResponseSuccess:      mts.Counter(ResponseSucc),
		ResponseFail:         mts.Counter(ResponseFail),
	}
	statsFactory[key] = s
	return s
}
