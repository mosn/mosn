package cluster

import (
	"context"
	"math/rand"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
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
	return &randomloadbalancer{}
}

func (l *randomloadbalancer) ChooseHost(context context.Context) types.Host {
	// TODO: randomly select priority
	hostset := l.prioritySet.HostSetsByPriority()[0]
	hosts := hostset.HealthyHosts()

	random := rand.Rand{}
	hostIdx := random.Int() % len(hosts)

	return hosts[hostIdx]
}

// TODO: more loadbalancers
