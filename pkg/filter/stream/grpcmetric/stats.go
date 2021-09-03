package grpcmetric

import (
	"sync"

	gometrics "github.com/rcrowley/go-metrics"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
)

var (
	serviceReqNum = "request_total"
	responseSucc  = "response_succ_total"
	responseFail  = "response_fail_total"
	serviceKey    = "service"
	metricPre     = "grpc"
)

type stats struct {
	requestServiceTootle gometrics.Counter
	responseSuccess      gometrics.Counter
	responseFail         gometrics.Counter
}

type state struct {
	mux          sync.RWMutex
	statsFactory map[string]*stats
}

func newState() *state {
	return &state{statsFactory: make(map[string]*stats)}
}

func (s *state) getStats(service string) *stats {
	key := service
	s.mux.RLock()
	stat, ok := s.statsFactory[key]
	s.mux.RUnlock()
	if ok {
		return stat
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	if stat, ok = s.statsFactory[key]; ok {
		return stat
	}
	labels := map[string]string{
		serviceKey: key,
	}
	mts, err := metrics.NewMetrics(metricPre, labels)
	if err != nil {
		log.DefaultLogger.Errorf("create metrics fail: labels:%v, err: %v", labels, err)
		s.statsFactory[key] = nil
		return nil
	}

	stat = &stats{
		requestServiceTootle: mts.Counter(serviceReqNum),
		responseSuccess:      mts.Counter(responseSucc),
		responseFail:         mts.Counter(responseFail),
	}
	s.statsFactory[key] = stat
	return stat
}
