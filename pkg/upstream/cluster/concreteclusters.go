package cluster

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"net"
)

type dynamicClusterBase struct {
	cluster
}

func (dc *dynamicClusterBase) updateDynamicHostList(newHosts []types.Host, currentHosts []types.Host) (
	changed bool, finalHosts []types.Host, hostsAdded []types.Host, hostsRemoved []types.Host) {
	hostAddrs := make(map[string]bool)

	// N^2 loop, works for small and steady hosts
	for _, nh := range newHosts {
		nhAddr := nh.AddressString()
		if _, ok := hostAddrs[nhAddr]; ok {
			continue
		}

		hostAddrs[nhAddr] = true

		found := false
		for i := 0; i < len(currentHosts); {
			curNh := currentHosts[i]

			if nh.AddressString() == curNh.AddressString() {
				curNh.SetWeight(nh.Weight())
				finalHosts = append(finalHosts, curNh)
				currentHosts = append(currentHosts[:i], currentHosts[i+1:]...)
				found = true
			} else {
				i++
			}
		}

		if !found {
			finalHosts = append(finalHosts, nh)
			hostsAdded = append(hostsAdded, nh)
		}
	}

	if len(currentHosts) > 0 {
		hostsRemoved = currentHosts
	}

	if len(hostsAdded) > 0 || len(hostsRemoved) > 0 {
		changed = true
	} else {
		changed = false
	}

	return changed, finalHosts, hostsAdded, hostsRemoved
}

// SimpleCluster
type simpleInMemCluster struct {
	dynamicClusterBase

	hosts []types.Host
}

func newSimpleInMemCluster(clusterConfig v2.Cluster, sourceAddr net.Addr, addedViaApi bool) *simpleInMemCluster {
	cluster := newCluster(clusterConfig, sourceAddr, addedViaApi, nil)

	return &simpleInMemCluster{
		dynamicClusterBase: dynamicClusterBase{
			cluster: cluster,
		},
	}
}

func (sc *simpleInMemCluster) UpdateHosts(newHosts []types.Host) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	
	var curHosts = make([]types.Host,len(sc.hosts))
	
	copy(curHosts, sc.hosts)
	changed, finalHosts, hostsAdded, hostsRemoved := sc.updateDynamicHostList(newHosts, curHosts)
	
	if len(finalHosts) == 0 {
		log.DefaultLogger.Debugf("final host is []")
	}
	
	for i, f := range finalHosts {
		log.DefaultLogger.Debugf("final host index = %d, address = %s,",i,f.AddressString())
	}
	
	log.DefaultLogger.Debugf("changed %s", changed)
	
	if changed {
		sc.hosts = finalHosts
		// todo: need to consider how to update healthyHost
		// Note: currently, we only use priority 0
		sc.prioritySet.GetOrCreateHostSet(0).UpdateHosts(sc.hosts,
			sc.hosts, nil, nil, hostsAdded, hostsRemoved)
		
		if sc.healthChecker != nil {
			sc.healthChecker.OnClusterMemberUpdate(hostsAdded,hostsRemoved)
		}
	}
	
	if len(sc.hosts) == 0 {
		log.DefaultLogger.Debugf(" after update final host is []")
	}
	
	for i, f := range sc.hosts {
		log.DefaultLogger.Debugf("after update final host index = %d, address = %s,",i,f.AddressString())
	}
}