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
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dchest/siphash"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/module/segmenttree"
	"mosn.io/mosn/pkg/variable"

	"github.com/trainyao/go-maglev"
	"mosn.io/api"
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
	RegisterLBType(types.WeightedRoundRobin, newSmoothWeightedRRLoadBalancer)
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
SW (smoothWeightedRRLoadBalancer) is a struct that contains weighted items and provides methods to select a weighted item.
It is used for the smooth weighted round-robin balancing algorithm. This algorithm is implemented in Nginx:
https://github.com/phusion/nginx/commit/27e94984486058d73157038f7950a0a36ecc6e35.
Algorithm is as follows: on each peer selection we increase current_weight
of each eligible peer by its weight, select peer with greatest current_weight
and reduce its current_weight by total number of weight points distributed
among peers.
*/
type smoothWeightedRRLoadBalancer struct {
	hosts         types.HostSet
	hostsWeighted []*hostSmoothWeighted
	lock          sync.Mutex
}

type hostSmoothWeighted struct {
	weight          int64
	currentWeight   int64
	effectiveWeight int64
}

func newSmoothWeightedRRLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	smoothWRRLoadBalancer := &smoothWeightedRRLoadBalancer{
		hosts:         hosts,
		hostsWeighted: make([]*hostSmoothWeighted, len(hosts.Hosts())),
	}
	// iterate over all hosts to init host with Weighted
	for idx, host := range hosts.Hosts() {
		w := int64(host.Weight())
		smoothWRRLoadBalancer.hostsWeighted[idx] = &hostSmoothWeighted{
			effectiveWeight: w,
			weight:          w,
		}
	}
	return smoothWRRLoadBalancer
}

// smooth weighted round robin
// O(n), traverse over all hosts
func (lb *smoothWeightedRRLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	var totalWeight int64 = 0
	var selectedHostWeighted *hostSmoothWeighted
	var selectedHost types.Host
	for idx, host := range lb.hosts.Hosts() {
		if !host.Health() {
			continue
		}
		hw := lb.hostsWeighted[idx]
		atomic.AddInt64(&hw.currentWeight, atomic.LoadInt64(&hw.effectiveWeight))
		totalWeight += atomic.LoadInt64(&hw.effectiveWeight)

		if hw.effectiveWeight < hw.weight {
			atomic.AddInt64(&hw.effectiveWeight, 1)
		}

		if selectedHostWeighted == nil || atomic.LoadInt64(&hw.currentWeight) > atomic.LoadInt64(&selectedHostWeighted.currentWeight) {
			selectedHostWeighted = hw
			selectedHost = host
		}
	}
	//
	if selectedHostWeighted != nil {
		atomic.AddInt64(&selectedHostWeighted.currentWeight, -totalWeight)
	}
	return selectedHost
}

func (lb *smoothWeightedRRLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return len(lb.hosts.Hosts()) > 0
}

func (lb *smoothWeightedRRLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
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

func newMaglevLoadBalancer(info types.ClusterInfo, set types.HostSet) types.LoadBalancer {
	var table *maglev.Table
	var tree *segmenttree.Tree
	names := []string{}
	for _, host := range set.Hosts() {
		names = append(names, host.Hostname())
	}
	if len(names) != 0 {
		table = maglev.New(names, maglev.SmallM)
		// build tree, devide hash
		nodes := []segmenttree.Node{}
		step := math.MaxUint64 / uint64(len(names))
		for index := range names {
			nodes = append(nodes, segmenttree.Node{
				Value:      index,
				RangeStart: uint64(index) * step,
				RangeEnd:   uint64(index+1) * step,
			})
		}
		updateFunc := func(lv, rv interface{}) interface{} {
			if lv != nil {
				leftIndex, ok := lv.(int)
				if !ok {
					return nil
				}
				if set.Hosts()[leftIndex].Health() {
					return leftIndex
				}
			}

			if rv != nil {
				rightIndex, ok := rv.(int)
				if !ok {
					return nil
				}

				if set.Hosts()[rightIndex].Health() {
					return rightIndex
				}
			}

			return nil
		}

		tree = segmenttree.NewTree(nodes, updateFunc)
	}

	mgv := &maglevLoadBalancer{
		hosts:           set,
		maglev:          table,
		fallbackSegTree: tree,
	}

	return mgv
}

type maglevLoadBalancer struct {
	hosts           types.HostSet
	maglev          *maglev.Table
	fallbackSegTree *segmenttree.Tree
}

func (lb *maglevLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	// host empty, maglev info may be nil
	if lb.maglev == nil {
		return nil
	}

	ch := context.ConsistentHashCriteria()
	if ch == nil || ch.HashType() != api.Maglev {
		return nil
	}

	hash := lb.generateChooseHostHash(context, ch)
	index := lb.maglev.Lookup(hash)
	chosen := lb.hosts.Hosts()[index]

	// fallback
	if !chosen.Health() {
		chosen = lb.chooseHostFromSegmentTree(index)
	}

	log.Proxy.Debugf(nil, "[lb][maglev] get index %d host %s %s",
		index, chosen.Hostname(), chosen.AddressString())

	return chosen
}

func (lb *maglevLoadBalancer) generateChooseHostHash(context types.LoadBalancerContext, info api.ConsistentHashCriteria) uint64 {
	switch info.(type) {
	case *v2.HeaderHashPolicy:
		headerKey := info.(*v2.HeaderHashPolicy).Key
		protocolVarHeaderKey := fmt.Sprintf("%s%s", types.VarProtocolRequestHeader, headerKey)

		headerValue, err := variable.GetProtocolResource(context.DownstreamContext(), api.HEADER, protocolVarHeaderKey)

		if err == nil {
			hashString := fmt.Sprintf("%s:%s", headerKey, headerValue)
			hash := getHashByString(hashString)
			return hash
		}
	case *v2.SourceIPHashPolicy:
		return getHashByAddr(context.DownstreamConnection().RemoteAddr())
	case *v2.HttpCookieHashPolicy:
		info := info.(*v2.HttpCookieHashPolicy)
		cookieName := info.Name
		protocolVarKey := fmt.Sprintf("%s%s", types.VarProtocolCookie, cookieName)

		cookieValue, err := variable.GetProtocolResource(context.DownstreamContext(), api.COOKIE, protocolVarKey)
		if err == nil {
			h := getHashByString(fmt.Sprintf("%s=%s", cookieName, cookieValue))
			return h
		}
	default:
	}

	return 0
}

func (lb *maglevLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.HostNum(metadata) > 0
}

func (lb *maglevLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return len(lb.hosts.Hosts())
}

func (lb *maglevLoadBalancer) chooseHostFromSegmentTree(index int) types.Host {
	if lb.fallbackSegTree == nil {
		return nil
	}

	leaf, err := lb.fallbackSegTree.Leaf(index)
	if err != nil {
		log.DefaultLogger.Errorf("[proxy] [maglev] [segmenttree] find leaf of index %d failed, err:%+v", index, err)
		return nil
	}

	// update tree value when
	lb.fallbackSegTree.Update(leaf)

	// leaf already unhealthy, find parent for it
	leaf = lb.fallbackSegTree.FindParent(leaf)
	var host types.Host
	for {
		if leaf.Value != nil {
			hostIndex, ok := leaf.Value.(int)
			if ok {
				if lb.hosts.Hosts()[hostIndex].Health() {
					host = lb.hosts.Hosts()[hostIndex]
					break
				}
			}
		}

		if leaf.IsRoot() {
			break
		}

		leaf = lb.fallbackSegTree.FindParent(leaf)
	}

	return host
}

func getHashByAddr(addr net.Addr) (hash uint64) {
	if tcpaddr, ok := addr.(*net.TCPAddr); ok {
		if len(tcpaddr.IP) == 16 || len(tcpaddr.IP) == 4 {
			var tmp uint32

			if len(tcpaddr.IP) == 16 {
				tmp = binary.BigEndian.Uint32(tcpaddr.IP[12:16])
			} else {
				tmp = binary.BigEndian.Uint32(tcpaddr.IP)
			}
			hash = uint64(tmp)

			return
		}
	}

	return getHashByString(fmt.Sprintf("%s", addr.String()))
}

func getHashByString(str string) uint64 {
	return siphash.Hash(0xbeefcafebabedead, 0, []byte(str))
}
