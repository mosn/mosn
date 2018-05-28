package cluster

import (
	"context"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"math/rand"
)

func NewLoadBalancer(lbType types.LoadBalancerType, prioritySet types.PrioritySet) types.LoadBalancer {
	switch lbType {
	case types.Random:
		return newRandomLoadbalancer(prioritySet)
	}

	return nil
}

type loadbalaner struct {
	prioritySet types.PrioritySet
}

// Random LoadBalancer
type randomloadbalancer struct {
	loadbalaner
}

func newRandomLoadbalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	return &randomloadbalancer{
		loadbalaner: loadbalaner{
			prioritySet: prioritySet,
		},
	}
}

func (l *randomloadbalancer) ChooseHost(context context.Context) types.Host {
	hostSets := l.prioritySet.HostSetsByPriority()
	idx := rand.Intn(len(hostSets))
	hostset := hostSets[idx]

	hosts := hostset.HealthyHosts()
	logger := log.ByContext(context)
	
	if len(hosts) == 0 {
		logger.Debugf("Choose host failed, no health host found")
		return nil
	}
	hostIdx := rand.Intn(len(hosts))
	return hosts[hostIdx]
}

// TODO: more loadbalancers@boqin
