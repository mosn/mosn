package cluster

import (
"context"
"math/rand"


"gitlab.alipay-inc.com/afe/mosn/pkg/log"
"gitlab.alipay-inc.com/afe/mosn/pkg/types"


)

func NewLoadBalancer(lbType types.LoadBalancerType, prioritySet types.PrioritySet) types.LoadBalancer {
	switch lbType {
	case types.Random:
		return newRandomLoadbalancer(prioritySet)
	case types.RoundRobin:
		return newRoundRobinLoadBalancer(prioritySet)
	}

	return nil
}

type loadbalaner struct {
	prioritySet types.PrioritySet
}

// Random LoadBalancer
type randomLoadBalancer struct {
	loadbalaner
}

func newRandomLoadbalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	return &randomLoadBalancer{
		loadbalaner: loadbalaner{
			prioritySet: prioritySet,
		},
	}
}

func (l *randomLoadBalancer) ChooseHost(context context.Context) types.Host {
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
type roundRobinLoadBalancer struct {
	loadbalaner
	// rrIndex for hostSet select
	rrIndexPriority uint32
	// rrInde for host select
	rrIndex         uint32
}

func newRoundRobinLoadBalancer(prioritySet types.PrioritySet) types.LoadBalancer {
	return &roundRobinLoadBalancer{
		loadbalaner: loadbalaner{
			prioritySet: prioritySet,
		},
	}
}

func (l *roundRobinLoadBalancer) ChooseHost(context context.Context) types.Host {
	var selectedHostSet []types.Host
	
	hostSets := l.prioritySet.HostSetsByPriority()
	hostSetsNum := uint32(len(hostSets))
	curHostSet := hostSets[l.rrIndexPriority % hostSetsNum].HealthyHosts()
	
	if l.rrIndex >= uint32(len(curHostSet)) {
		l.rrIndexPriority = (l.rrIndexPriority + 1) % hostSetsNum
		l.rrIndex = 0
		selectedHostSet = hostSets[l.rrIndexPriority].HealthyHosts()
	} else {
		selectedHostSet = curHostSet
	}
	
	if len(selectedHostSet) == 0 {
		logger := log.ByContext(context)
		logger.Debugf("Choose host in RoundRobin failed, no health host found")
		return nil
	}
	
	selectedHost := selectedHostSet[l.rrIndex % uint32(len(selectedHostSet))]
	l.rrIndex ++
	
	return selectedHost
}