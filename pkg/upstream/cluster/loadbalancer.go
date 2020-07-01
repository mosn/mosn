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

package cluster

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trainyao/go-maglev"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// NewLoadBalancer can be register self defined type
var lbFactories map[types.LoadBalancerType]func(types.ClusterInfo, types.HostSet) types.LoadBalancer

func RegisterLBType(lbType types.LoadBalancerType, f func(types.ClusterInfo, types.HostSet) types.LoadBalancer) {
	if lbFactories == nil {
		lbFactories = make(map[types.LoadBalancerType]func(types.ClusterInfo, types.HostSet) types.LoadBalancer)
	}
	lbFactories[lbType] = f
}

var rrFactory *roundRobinLoadBalancerFactory

func init() {
	rrFactory = &roundRobinLoadBalancerFactory{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	RegisterLBType(types.RoundRobin, rrFactory.newRoundRobinLoadBalancer)
	RegisterLBType(types.Random, newRandomLoadBalancer)
	RegisterLBType(types.WeightedRoundRobin, newWRRLoadBalancer)
	RegisterLBType(types.LeastActiveRequest, newleastActiveRequestLoadBalancer)
	RegisterLBType(types.Maglev, newMaglevLoadBalancer)
}

func NewLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	lbType := info.LbType()
	if f, ok := lbFactories[lbType]; ok {
		return f(info, hosts)
	}
	return rrFactory.newRoundRobinLoadBalancer(info, hosts)
}

// LoadBalancer Implementations

type randomLoadBalancer struct {
	mutex sync.Mutex
	rand  *rand.Rand
	hosts types.HostSet
	rrLB  types.LoadBalancer // if node fails, we'll degrade to rr load balancer
}

func newRandomLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	return &randomLoadBalancer{
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		hosts: hosts,
		rrLB:  rrFactory.newRoundRobinLoadBalancer(info, hosts),
	}
}

func (lb *randomLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	targets := lb.hosts.Hosts()
	total := len(targets)
	if total == 0 {
		return nil
	}

	lb.mutex.Lock()
	idx := lb.rand.Intn(total)
	lb.mutex.Unlock()

	host := targets[idx]
	if host.Health() {
		return host
	}

	// degrade to rr lb, to make node selection more balanced
	return lb.rrLB.ChooseHost(context)
}

func (lb *randomLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *randomLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

type roundRobinLoadBalancer struct {
	hosts   types.HostSet
	rrIndex uint32
}

type roundRobinLoadBalancerFactory struct {
	mutex sync.Mutex
	rand  *rand.Rand
}

func (f *roundRobinLoadBalancerFactory) newRoundRobinLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	var idx uint32
	hostsList := hosts.Hosts()
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if len(hostsList) != 0 {
		idx = f.rand.Uint32() % uint32(len(hostsList))
	}
	return &roundRobinLoadBalancer{
		hosts:   hosts,
		rrIndex: idx,
	}
}

func (lb *roundRobinLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	targets := lb.hosts.Hosts()
	total := len(targets)
	if total == 0 {
		return nil
	}
	for i := 0; i < total; i++ {
		index := atomic.AddUint32(&lb.rrIndex, 1) % uint32(total)
		host := targets[index]
		if host.Health() {
			return host
		}
	}
	return nil
}

func (lb *roundRobinLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *roundRobinLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

/*
 A round robin load balancer. When in weighted mode, EDF scheduling is used. When in not
 weighted mode, simple RR index selection is used.
*/
type WRRLoadBalancer struct {
	*EdfLoadBalancer
	rrIndex uint32
}

func newWRRLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	wrrLB := &WRRLoadBalancer{}
	wrrLB.EdfLoadBalancer = newEdfLoadBalancerLoadBalancer(hosts, wrrLB.unweightChooseHost, wrrLB.hostWeight)
	hostsList := hosts.Hosts()
	wrrLB.mutex.Lock()
	defer wrrLB.mutex.Unlock()
	if len(hostsList) != 0 {
		wrrLB.rrIndex = wrrLB.rand.Uint32() % uint32(len(hostsList))
	}
	return wrrLB
}

func (lb *WRRLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *WRRLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

func (lb *WRRLoadBalancer) hostWeight(item WeightItem) float64 {
	host := item.(types.Host)
	return float64(host.Weight())
}

// do unweighted (fast) selection
func (lb *WRRLoadBalancer) unweightChooseHost(context types.LoadBalancerContext) types.Host {
	targets := lb.hosts.Hosts()
	total := len(targets)
	index := atomic.AddUint32(&lb.rrIndex, 1) % uint32(total)
	return targets[index]
}

const default_choice = 2

// leastActiveRequestLoadBalancer choose the host with the least active request
type leastActiveRequestLoadBalancer struct {
	*EdfLoadBalancer
	choice uint32
}

func newleastActiveRequestLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	lb := &leastActiveRequestLoadBalancer{}
	if info != nil && info.LbConfig() != nil {
		lb.choice = info.LbConfig().(*v2.LeastRequestLbConfig).ChoiceCount
	} else {
		lb.choice = default_choice
	}
	lb.EdfLoadBalancer = newEdfLoadBalancerLoadBalancer(hosts, lb.unweightChooseHost, lb.hostWeight)
	return lb
}

func (lb *leastActiveRequestLoadBalancer) hostWeight(item WeightItem) float64 {
	host := item.(types.Host)
	return float64(host.Weight()) / float64(host.HostStats().UpstreamRequestActive.Count()+1)
}

func (lb *leastActiveRequestLoadBalancer) unweightChooseHost(context types.LoadBalancerContext) types.Host {

	allHosts := lb.hosts.Hosts()
	total := len(allHosts)
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	var candicate types.Host
	// Choose `choice` times and return the best one
	// See The Power of Two Random Choices: A Survey of Techniques and Results
	//  http://www.eecs.harvard.edu/~michaelm/postscripts/handbook2001.pdf
	for cur := 0; cur < int(lb.choice); cur++ {

		randIdx := lb.rand.Intn(total)
		tempHost := allHosts[randIdx]
		if candicate == nil {
			candicate = tempHost
			continue
		}
		if candicate.HostStats().UpstreamRequestActive.Count() > tempHost.HostStats().UpstreamRequestActive.Count() {
			candicate = tempHost
		}
	}
	return candicate

}

type EdfLoadBalancer struct {
	scheduler *edfSchduler
	hosts     types.HostSet
	rand      *rand.Rand
	mutex     sync.Mutex
	// the method to choose host when all host
	unweightChooseHostFunc func(types.LoadBalancerContext) types.Host
	hostWeightFunc         func(item WeightItem) float64
}

func (lb *EdfLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {

	var candicate types.Host
	targetHosts := lb.hosts.Hosts()
	total := len(targetHosts)
	if total == 0 {
		// Return nil directly if allHosts is nil or size is 0
		return nil
	}
	if total == 1 {
		// Return directly if there is only one host
		return targetHosts[0]
	}
	for i := 0; i < total; i++ {
		if lb.scheduler != nil {
			// do weight selection
			candicate = lb.scheduler.NextAndPush(lb.hostWeightFunc).(types.Host)
		} else {
			// do unweight selection
			candicate = lb.unweightChooseHostFunc(context)
		}
		// only return when candicate is healthy
		if candicate.Health() {
			return candicate
		}
	}
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	// randomly choose one when all instances are unhealthy
	return targetHosts[lb.rand.Intn(total)]
}

func (lb *EdfLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *EdfLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

func newEdfLoadBalancerLoadBalancer(hosts types.HostSet, unWeightChoose func(types.LoadBalancerContext) types.Host, hostWeightFunc func(host WeightItem) float64) *EdfLoadBalancer {
	lb := &EdfLoadBalancer{
		hosts:                  hosts,
		rand:                   rand.New(rand.NewSource(time.Now().UnixNano())),
		unweightChooseHostFunc: unWeightChoose,
		hostWeightFunc:         hostWeightFunc,
	}
	lb.refresh(hosts.Hosts())
	return lb
}

func (lb *EdfLoadBalancer) refresh(hosts []types.Host) {
	// Check if the original host weights are equal and skip EDF creation if they are
	if hostWeightsAreEqual(hosts) {
		return
	}

	lb.scheduler = newEdfScheduler(len(hosts))

	// Init Edf scheduler with healthy hosts.
	for _, host := range hosts {
		lb.scheduler.Add(host, lb.hostWeightFunc(host))
	}

}

func hostWeightsAreEqual(hosts []types.Host) bool {
	if len(hosts) <= 1 {
		return true
	}
	weight := hosts[0].Weight()

	for i := 1; i < len(hosts); i++ {
		if hosts[i].Weight() != weight {
			return false
		}
	}
	return true
}

// newMaglevLoadBalancer return maglevLoadBalancer structure.
//
// In maglevLoadBalancer, there is a maglev table for consistence hash host choosing.
// If the chosen host is unhealthy, maglevLoadBalancer will traverse host list to find a healthy host.
func newMaglevLoadBalancer(info types.ClusterInfo, set types.HostSet) types.LoadBalancer {
	names := []string{}
	for _, host := range set.Hosts() {
		names = append(names, host.AddressString())
	}
	mgv := &maglevLoadBalancer{
		hosts: set,
	}

	nameCount := len(names)
	// if host count > BigM, maglev table building will cross array boundary
	// maglev lb will not work in this scenario
	if nameCount >= maglev.BigM {
		log.DefaultLogger.Errorf("[lb][maglev] host count too large, expect <= %d, get %d",
			maglev.BigM, nameCount)
		return mgv
	}
	if nameCount == 0 {
		return mgv
	}

	maglevM := maglev.SmallM
	// according to test, 30000 host with testing 1e8 times, hash distribution begins to go wrong,
	// max=4855, mean=3333.3333333333335, peak-to-mean=1.4565
	// so use BigM when host >= 30000
	limit := 30000
	if nameCount >= limit {
		log.DefaultLogger.Infof("[lb][maglev] host count %d >= %d, using maglev.BigM", nameCount, limit)
		maglevM = maglev.BigM
	}

	mgv.maglev = maglev.New(names, uint64(maglevM))
	return mgv
}

type maglevLoadBalancer struct {
	hosts  types.HostSet
	maglev *maglev.Table
}

func (lb *maglevLoadBalancer) ChooseHost(ctx types.LoadBalancerContext) types.Host {
	// host empty, maglev info may be nil
	if lb.maglev == nil {
		return nil
	}

	route := ctx.DownstreamRoute()
	if route == nil || route.RouteRule() == nil {
		return nil
	}

	hashPolicy := route.RouteRule().Policy().HashPolicy()
	if hashPolicy == nil {
		return nil
	}

	hash := hashPolicy.GenerateHash(ctx.DownstreamContext())
	index := lb.maglev.Lookup(hash)
	chosen := lb.hosts.Hosts()[index]

	// fallback
	if !chosen.Health() {
		chosen = lb.chooseHostFromHostList(index)
	}

	if chosen == nil {
		log.Proxy.Infof(ctx.DownstreamContext(), "[lb][maglev] hash %d get nil host, index: %d",
			hash, index)
	} else {
		log.Proxy.Debugf(ctx.DownstreamContext(), "[lb][maglev] hash %d index %d get host %s",
			hash, index, chosen.AddressString())
	}

	return chosen
}

func (lb *maglevLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.HostNum(metadata) > 0
}

func (lb *maglevLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

// chooseHostFromHostList traverse host list to find a healthy host
func (lb *maglevLoadBalancer) chooseHostFromHostList(index int) types.Host {
	hostCount := len(lb.hosts.Hosts())

	// go left
	counterIndex := index
	for counterIndex > 0 {
		counterIndex--

		if lb.hosts.Hosts()[counterIndex].Health() {
			return lb.hosts.Hosts()[counterIndex]
		}
	}

	// go right
	for counterIndex = index + 1; counterIndex < hostCount; counterIndex++ {
		if lb.hosts.Hosts()[counterIndex].Health() {
			return lb.hosts.Hosts()[counterIndex]
		}
	}

	return nil
}
