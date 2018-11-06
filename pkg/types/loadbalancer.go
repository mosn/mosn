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

package types

import (
	"net"
)

// LoadBalancerType is the load balancer's type
type LoadBalancerType string

// The load balancer's types
const (
	RoundRobin LoadBalancerType = "RoundRobin"
	Random     LoadBalancerType = "Random"
)

// LoadBalancer is a upstream load balancer.
// When a request comes, the LoadBalancer will choose a upstream cluster's host to handle the request.
type LoadBalancer interface {
	// ChooseHost chooses a host based on the load balancer context
	ChooseHost(context LoadBalancerContext) Host
}

// LoadBalancerContext contains the information for choose a host
type LoadBalancerContext interface {
	// ComputeHashKey computes an optional hash key to use during load balancing
	ComputeHashKey() HashedValue

	// MetadataMatchCriteria gets metadata match criteria used for selecting a subset of hosts
	MetadataMatchCriteria() MetadataMatchCriteria

	// DownstreamConnection returns the downstream connection.
	DownstreamConnection() net.Conn

	// DownstreamHeaders returns the downstream headers map.
	DownstreamHeaders() HeaderMap
}

// SubSetLoadBalancer is a subset of LoadBalancer
type SubSetLoadBalancer interface {
	LoadBalancer

	// TryChooseHostFromContext finds a host from the subsets, context is LoadBalancerContext.
	// The bool returned true if a Host returned is not nil.
	// If no metadata found in the context, or there is no matching subset, or the matching subset
	// contains no available host will return a nil Host and false.
	TryChooseHostFromContext(context LoadBalancerContext) (Host, bool)

	// UpdateFallbackSubset creates and updates fallback subset.
	UpdateFallbackSubset(priority uint32, hostAdded []Host, hostsRemoved []Host)

	// HostMatches returns the host is match or not.
	HostMatches(kvs SubsetMetadata, host Host) bool

	// FindSubset iterates over the given metadata match criteria (which must be lexically sorted by key) and find
	// a matching LbSubsetEntryPtr, if any.
	FindSubset(matches []MetadataMatchCriterion) LBSubsetEntry

	// FindOrCreateSubset recursively finds the matching LbSubsetEntry by a vector of key-values (from extractSubsetMetadata)
	FindOrCreateSubset(subsets LbSubsetMap, kvs SubsetMetadata, idx uint32) LBSubsetEntry

	// ExtractSubsetMetadata iterates over subset_keys looking up values from the given host's metadata. Each key-value pair
	// is appended to kvs. Returns a non-empty value if the host has a value for each key.
	ExtractSubsetMetadata(subsetKeys []string, host Host) SubsetMetadata

	// Update updates or creates subsets for one priority level.
	Update(priority uint32, hostAdded []Host, hostsRemoved []Host)

	// ProcessSubsets is called when host added or removed, to:
	// update lbsubset entry by updateCB or
	// create lbsubset entry by newCB
	ProcessSubsets(hostAdded []Host, hostsRemoved []Host,
		updateCB func(LBSubsetEntry), newCB func(LBSubsetEntry, HostPredicate, SubsetMetadata, bool))
}

// LBSubsetEntry is a entry that stored in the subset hierarchy.
type LBSubsetEntry interface {
	// Initialized returns the entry is initialized or not.
	Initialized() bool

	// Active returns the entry is active or not.
	Active() bool

	// PrioritySubset returns the tored subset matched.
	PrioritySubset() PrioritySubset

	// SetPrioritySubset sets the entry's priority subset.
	SetPrioritySubset(PrioritySubset)

	// Children returns the next lb subset map
	Children() LbSubsetMap
}

// HostSubset represents a subset of an original HostSet.
type HostSubset interface {
	// UpdateHostSubset updates the host subset.
	UpdateHostSubset(hostsAdded []Host, hostsRemoved []Host, predicate HostPredicate)

	// Empty returns HostSubset is empty or not.
	Empty() bool

	//Hosts returns the Host list.
	Hosts() []Host

	//TODO:
	// TriggerCallbacks()
}

// PrioritySubset represents a subset of an original PrioritySet.
type PrioritySubset interface {
	// Update updates priority subset.
	Update(priority uint32, hostsAdded []Host, hostsRemoved []Host)

	// Empty returns the priority subset is empty or not.
	Empty() bool

	// GetOrCreateHostSubset returns a host subset which matches the given priority.
	// A new host subset is created if there is no priority host subset matched.
	GetOrCreateHostSubset(priority uint32) HostSubset

	// TriggerCallbacks triggers the callback functions.
	TriggerCallbacks()

	// CreateHostSet creates a host set
	CreateHostSet(priority uint32) HostSet

	// LB returns the LoadBalancer
	LB() LoadBalancer
}

// FallBackPolicy type
type FallBackPolicy uint8

// FallBackPolicy types
const (
	NoFallBack FallBackPolicy = iota
	AnyEndPoint
	DefaultSubsetDefaultSubset
)

// LbSubsetMap is a trie-like structure. Route Metadata requires lexically sorted
// act as the root.
type LbSubsetMap map[string]ValueSubsetMap

// ValueSubsetMap is a LBSubsetEntry map.
type ValueSubsetMap map[HashedValue]LBSubsetEntry

//type LBSubsetEntry struct {
//	children       types.LbSubsetMap
//	prioritySubset types.PrioritySubset
//}

// SubsetMetadata is a vector of key-values
type SubsetMetadata []Pair

// Pair is a key-value pair that contains metadata.
type Pair struct {
	T1 string
	T2 HashedValue
}

// ResourcePriority type
type ResourcePriority uint8

// ResourcePriority types
const (
	Default ResourcePriority = 0
	High    ResourcePriority = 1
)
