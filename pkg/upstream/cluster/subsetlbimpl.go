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

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
)

type subSetLoadBalancer struct {
	lbType                types.LoadBalancerType // inner LB algorithm for choosing subset's host
	runtime               types.Loader
	stats                 types.ClusterStats
	random                rand.Rand
	fallBackPolicy        types.FallBackPolicy
	defaultSubSetMetadata types.SubsetMetadata        // default subset metadata
	subSetKeys            []types.SortedStringSetType // subset selectors
	originalPrioritySet   types.PrioritySet
	fallbackSubset        *LBSubsetEntry    // subset entry generated according to fallback policy
	subSets               types.LbSubsetMap // final trie-like structure used to stored easily searched subset
}

//
func NewSubsetLoadBalancer(lbType types.LoadBalancerType, prioritySet types.PrioritySet, stats types.ClusterStats,
	subsets types.LBSubsetInfo) types.SubSetLoadBalancer {

	ssb := &subSetLoadBalancer{
		lbType:                lbType,
		fallBackPolicy:        subsets.FallbackPolicy(),
		defaultSubSetMetadata: GenerateDftSubsetKeys(subsets.DefaultSubset()), //ordered subset metadata pair, value为md5 hash值
		subSetKeys:            subsets.SubsetKeys(),
		originalPrioritySet:   prioritySet,
		stats:                 stats,
	}

	ssb.subSets = make(map[string]types.ValueSubsetMap)

	// foreach every priority subset
	// init subset, fallback subset and so on
	for _, hostSet := range prioritySet.HostSetsByPriority() {
		ssb.Update(hostSet.Priority(), hostSet.Hosts(), nil)
	}

	// add update callback when original priority set updated
	ssb.originalPrioritySet.AddMemberUpdateCb(
		func(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {
			ssb.Update(priority, hostsAdded, hostsRemoved)
		},
	)

	return ssb
}

// create or update subsets for this priority
func (sslb *subSetLoadBalancer) Update(priority uint32, hostAdded []types.Host, hostsRemoved []types.Host) {

	// step1. create or update fallback subset
	sslb.UpdateFallbackSubset(priority, hostAdded, hostsRemoved)

	// step2. create or update global subset
	sslb.ProcessSubsets(hostAdded, hostsRemoved,
		func(entry types.LBSubsetEntry) {
			activeBefore := entry.Active()
			entry.PrioritySubset().Update(priority, hostAdded, hostsRemoved)

			if activeBefore && !entry.Active() {
				sslb.stats.LBSubSetsActive.Dec(1)
				sslb.stats.LBSubsetsRemoved.Inc(1)
			} else if !activeBefore && entry.Active() {
				sslb.stats.LBSubSetsActive.Inc(1)
				sslb.stats.LBSubsetsCreated.Inc(1)
			}
		},

		func(entry types.LBSubsetEntry, predicate types.HostPredicate, kvs types.SubsetMetadata, addinghost bool) {
			if addinghost {
				prioritySubset := NewPrioritySubsetImpl(sslb, predicate)
				log.DefaultLogger.Debugf("creating subset loadbalancing for %+v", kvs)
				entry.SetPrioritySubset(prioritySubset)
				sslb.stats.LBSubSetsActive.Inc(1)
				sslb.stats.LBSubsetsCreated.Inc(1)
			}
		})
}

// SubSet LB Entry
func (sslb *subSetLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {

	if nil != context {
		host, hostChoosen := sslb.TryChooseHostFromContext(context)
		if hostChoosen {
			log.DefaultLogger.Debugf("subset load balancer: match subset entry success, "+
				"choose hostaddr = %s", host.AddressString())
			return host
		}
	}

	if nil == sslb.fallbackSubset {
		log.DefaultLogger.Errorf("subset load balancer: failure, fallback subset is nil")
		return nil
	}
	sslb.stats.LBSubSetsFallBack.Inc(1)

	defaulthosts := sslb.fallbackSubset.prioritySubset.GetOrCreateHostSubset(0).Hosts()

	if len(defaulthosts) > 0 {
		log.DefaultLogger.Debugf("subset load balancer: use default subset,hosts are ", defaulthosts)
	} else {
		log.DefaultLogger.Errorf("subset load balancer: failure, fallback subset's host is nil")
		return nil
	}

	return sslb.fallbackSubset.prioritySubset.LB().ChooseHost(context)
}

func (sslb *subSetLoadBalancer) TryChooseHostFromContext(context types.LoadBalancerContext) (types.Host, bool) {

	matchCriteria := context.MetadataMatchCriteria()

	if nil == matchCriteria {
		log.DefaultLogger.Errorf("subset load balancer: context is nil")
		return nil, false
	}

	entry := sslb.FindSubset(matchCriteria.MetadataMatchCriteria())

	if nil == entry || !entry.Active() {
		log.DefaultLogger.Errorf("subset load balancer: match entry failure")
		return nil, false
	}

	return entry.PrioritySubset().LB().ChooseHost(context), true
}

// create or update fallback subset
func (sslb *subSetLoadBalancer) UpdateFallbackSubset(priority uint32, hostAdded []types.Host,
	hostsRemoved []types.Host) {

	if types.NoFallBack == sslb.fallBackPolicy {
		log.DefaultLogger.Debugf("subset load balancer: fallback is disabled")
		return
	}

	// create default host subset
	if nil == sslb.fallbackSubset {
		var predicate types.HostPredicate

		if types.AnyEndPoint == sslb.fallBackPolicy {
			predicate = func(types.Host) bool {
				return true
			}
		} else if types.DefaultSubsetDefaultSubset == sslb.fallBackPolicy {
			predicate = func(host types.Host) bool {
				return sslb.HostMatches(sslb.defaultSubSetMetadata, host)
			}
		}

		sslb.fallbackSubset = &LBSubsetEntry{
			children:       nil, // children is nil for fallback subset
			prioritySubset: NewPrioritySubsetImpl(sslb, predicate),
		}
	}

	// update default host subset
	sslb.fallbackSubset.prioritySubset.Update(priority, hostAdded, hostsRemoved)
}

// create or update subset
func (sslb *subSetLoadBalancer) ProcessSubsets(hostAdded []types.Host, hostsRemoved []types.Host,
	updateCB func(types.LBSubsetEntry), newCB func(types.LBSubsetEntry, types.HostPredicate, types.SubsetMetadata, bool)) {

	hostMapWithBool := map[bool][]types.Host{true: hostAdded, false: hostsRemoved}

	//	subSetsModified := mapset.NewSet(LBSubsetEntry{})
	for addinghost, hosts := range hostMapWithBool {

		for _, host := range hosts {

			for _, subSetKey := range sslb.subSetKeys {

				kvs := sslb.ExtractSubsetMetadata(subSetKey.Keys(), host)
				if len(kvs) > 0 {

					entry := sslb.FindOrCreateSubset(sslb.subSets, kvs, 0)

					if entry.Initialized() {
						updateCB(entry)
					} else {
						predicate := func(host types.Host) bool {
							return sslb.HostMatches(kvs, host)
						}

						newCB(entry, predicate, kvs, addinghost)
					}
				}
			}
		}
	}
}

// judge whether the host's metatada match the subset
// kvs and host must in the same order
func (sslb *subSetLoadBalancer) HostMatches(kvs types.SubsetMetadata, host types.Host) bool {
	hostMetadata := host.Metadata()

	for _, kv := range kvs {
		if value, ok := hostMetadata[kv.T1]; ok {
			if !types.EqualHashValue(value, kv.T2) {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

// search subset tree
func (sslb *subSetLoadBalancer) FindSubset(matchCriteria []types.MetadataMatchCriterion) types.LBSubsetEntry {
	// subsets : map[string]ValueSubsetMap
	// valueSubsetMap: map[HashedValue]LBSubsetEntry
	var subSets = sslb.subSets

	for i, mcCriterion := range matchCriteria {

		if vsMap, ok := subSets[mcCriterion.MetadataKeyName()]; ok {

			if vsEntry, ok := vsMap[mcCriterion.MetadataValue()]; ok {

				if i+1 == len(matchCriteria) {
					return vsEntry
				}

				subSets = vsEntry.Children()

			} else {
				break
			}
		} else {
			break
		}
	}

	return nil
}

// generate subset recursively
// return leaf node
func (sslb *subSetLoadBalancer) FindOrCreateSubset(subsets types.LbSubsetMap,
	kvs types.SubsetMetadata, idx uint32) types.LBSubsetEntry {

	if idx > uint32(len(kvs)) {
		log.DefaultLogger.Fatal("Find Or Create Subset Error")
	}

	name := kvs[idx].T1
	hashedValue := kvs[idx].T2
	var entry types.LBSubsetEntry

	if vsMap, ok := subsets[name]; ok {

		if lbEntry, ok := vsMap[hashedValue]; ok {
			// entry already exist
			entry = lbEntry
		} else {
			// new lbsubset entry
			entry = &LBSubsetEntry{
				children:       make(map[string]types.ValueSubsetMap),
				prioritySubset: nil,
			}
			vsMap[hashedValue] = entry
			subsets[name] = vsMap
		}
	} else {
		entry = &LBSubsetEntry{
			children:       make(map[string]types.ValueSubsetMap),
			prioritySubset: nil,
		}

		valueSubsetMap := types.ValueSubsetMap{
			hashedValue: entry,
		}

		subsets[name] = valueSubsetMap
	}

	idx++

	if idx == uint32(len(kvs)) {
		return entry
	}

	return sslb.FindOrCreateSubset(entry.Children(), kvs, idx)
}

// 从host的metadata 以及cluster的subset keys中，生成字典序的 subset metadata
// 之所以生成字典序是由于subsetkeys已经按照字典序排好了
// 生成规则：subsetkeys中所有的key均在host的metadata中出现

func (sslb *subSetLoadBalancer) ExtractSubsetMetadata(subsetKeys []string, host types.Host) types.SubsetMetadata {
	metadata := host.Metadata()
	var kvs types.SubsetMetadata

	for _, key := range subsetKeys {
		exist := false
		var value types.HashedValue

		for keyM, valueM := range metadata {
			if keyM == key {
				value = valueM
				exist = true
				break
			}
		}

		if !exist {
			break
		} else {
			v := types.Pair{key, value}
			kvs = append(kvs, v)
		}
	}

	if len(kvs) != len(subsetKeys) {
		kvs = []types.Pair{}
	}

	return kvs
}

type LBSubsetEntry struct {
	children       types.LbSubsetMap
	prioritySubset types.PrioritySubset
}

func (lbbe *LBSubsetEntry) Initialized() bool {
	return nil != lbbe.prioritySubset
}

func (lbbe *LBSubsetEntry) Active() bool {
	return true
}

func (lbbe *LBSubsetEntry) PrioritySubset() types.PrioritySubset {
	return lbbe.prioritySubset
}

func (lbbe *LBSubsetEntry) SetPrioritySubset(ps types.PrioritySubset) {
	lbbe.prioritySubset = ps
}

func (lbbe *LBSubsetEntry) Children() types.LbSubsetMap {
	return lbbe.children
}

type hostSubsetImpl struct {
	hostSubset types.HostSet
}

func (hsi *hostSubsetImpl) UpdateHostSubset(hostsAdded []types.Host, hostsRemoved []types.Host, predicate types.HostPredicate) {
	// todo check host predicate

	var filteredAdded []types.Host
	var filteredRemoved []types.Host

	//var hosts []types.Host
	var healthyHosts []types.Host

	for _, host := range hostsAdded {
		if predicate(host) {
			filteredAdded = append(filteredAdded, host)
		}
	}

	for _, host := range hostsRemoved {
		if predicate(host) {
			filteredRemoved = append(filteredRemoved, host)
		}
	}

	finalhosts := hsi.GetFinalHosts(filteredAdded, filteredRemoved)

	for _, host := range finalhosts {

		if host.Health() {
			healthyHosts = append(healthyHosts, host)
		}
	}

	//最终更新host
	hsi.hostSubset.UpdateHosts(finalhosts, healthyHosts, nil, nil,
		filteredAdded, filteredRemoved)
}

func (hsi *hostSubsetImpl) GetFinalHosts(hostsAdded []types.Host, hostsRemoved []types.Host) []types.Host {
	hosts := hsi.hostSubset.Hosts()

	for _, host := range hostsAdded {
		found := false
		for _, hostOrig := range hosts {
			if host.AddressString() == hostOrig.AddressString() {
				found = true
			}
		}

		if !found {
			hosts = append(hosts, host)
		}
	}

	for _, host := range hostsRemoved {
		for i, hostOrig := range hosts {
			if host.AddressString() == hostOrig.AddressString() {
				hosts = append(hosts[:i], hosts[i+1:]...)
				continue
			}
		}
	}

	return hosts
}

func (hsi *hostSubsetImpl) Empty() bool {
	return len(hsi.hostSubset.Hosts()) == 0
}

func (hsi *hostSubsetImpl) Hosts() []types.Host {
	return hsi.hostSubset.Hosts()
}

// subset of original priority set
type PrioritySubsetImpl struct {
	prioritySubset      types.PrioritySet // storing the matched host in subset
	originalPrioritySet types.PrioritySet
	empty               bool
	loadbalancer        types.LoadBalancer
	predicateFunc       types.HostPredicate // match function for host and metadata
}

func NewPrioritySubsetImpl(subsetLB *subSetLoadBalancer, predicate types.HostPredicate) types.PrioritySubset {
	psi := &PrioritySubsetImpl{
		originalPrioritySet: subsetLB.originalPrioritySet,
		predicateFunc:       predicate,
		empty:               true,
		prioritySubset:      &prioritySet{},
	}

	var i uint32

	for i = 0; i < uint32(len(psi.originalPrioritySet.HostSetsByPriority())); i++ {
		if len(psi.originalPrioritySet.GetOrCreateHostSet(i).Hosts()) > 0 {
			psi.empty = false
		}
	}

	for i = 0; i < uint32(len(psi.originalPrioritySet.HostSetsByPriority())); i++ {
		psi.Update(i, subsetLB.originalPrioritySet.HostSetsByPriority()[i].Hosts(), []types.Host{})
	}

	psi.loadbalancer = NewLoadBalancer(subsetLB.lbType, psi.prioritySubset)

	return psi
}

func (psi *PrioritySubsetImpl) Update(priority uint32, hostsAdded []types.Host, hostsRemoved []types.Host) {

	psi.GetOrCreateHostSubset(priority).UpdateHostSubset(hostsAdded, hostsRemoved, psi.predicateFunc)

	for _, hostSet := range psi.prioritySubset.HostSetsByPriority() {
		if len(hostSet.Hosts()) > 0 {
			psi.empty = false
			return
		}
	}
}

func (psi *PrioritySubsetImpl) Empty() bool {

	return psi.empty
}

func (psi *PrioritySubsetImpl) GetOrCreateHostSubset(priority uint32) types.HostSubset {
	return &hostSubsetImpl{
		hostSubset: psi.prioritySubset.GetOrCreateHostSet(priority),
	}
}

func (psi *PrioritySubsetImpl) TriggerCallbacks() {

}

func (psi *PrioritySubsetImpl) CreateHostSet(priority uint32) types.HostSet {
	return nil
}

func (psi *PrioritySubsetImpl) LB() types.LoadBalancer {
	return psi.loadbalancer
}

func NewLBSubsetInfo(subsetCfg *v2.LBSubsetConfig) types.LBSubsetInfo {
	lbSubsetInfo := &LBSubsetInfoImpl{
		fallbackPolicy: types.FallBackPolicy(subsetCfg.FallBackPolicy),
		subSetKeys:     GenerateSubsetKeys(subsetCfg.SubsetSelectors),
	}

	if len(subsetCfg.SubsetSelectors) == 0 {
		lbSubsetInfo.enabled = false
	} else {
		lbSubsetInfo.enabled = true
	}

	lbSubsetInfo.defaultSubSet = types.InitSortedMap(subsetCfg.DefaultSubset)

	return lbSubsetInfo
}

type LBSubsetInfoImpl struct {
	enabled        bool
	fallbackPolicy types.FallBackPolicy
	defaultSubSet  types.SortedMap             // sorted default subset
	subSetKeys     []types.SortedStringSetType // sorted subset selectors
}

func (lbsi *LBSubsetInfoImpl) IsEnabled() bool {
	return lbsi.enabled
}

func (lbsi *LBSubsetInfoImpl) FallbackPolicy() types.FallBackPolicy {
	return lbsi.fallbackPolicy
}

func (lbsi *LBSubsetInfoImpl) DefaultSubset() types.SortedMap {
	return lbsi.defaultSubSet
}

func (lbsi *LBSubsetInfoImpl) SubsetKeys() []types.SortedStringSetType {
	return lbsi.subSetKeys
}

// used to generate sorted keys
// without duplication
func GenerateSubsetKeys(keysArray [][]string) []types.SortedStringSetType {
	var ssst []types.SortedStringSetType

	for _, keys := range keysArray {
		sortedStringSet := types.InitSet(keys)
		found := false

		for _, s := range ssst {
			if judgeStringArrayEqual(sortedStringSet.Keys(), s.Keys()) {
				found = true
				break
			}
		}

		if !found {
			ssst = append(ssst, sortedStringSet)
		}
	}

	return ssst
}

func GenerateDftSubsetKeys(dftkeys types.SortedMap) types.SubsetMetadata {
	var sm types.SubsetMetadata
	for _, pair := range dftkeys {
		sm = append(sm, types.Pair{pair.Key, types.GenerateHashedValue(pair.Value)})
	}

	return sm
}

func judgeStringArrayEqual(T1 []string, T2 []string) bool {

	if len(T1) != len(T2) {
		return false
	}

	for i := 0; i < len(T1); i++ {
		if T1[i] != T2[i] {
			return false
		}
	}

	return true

}
