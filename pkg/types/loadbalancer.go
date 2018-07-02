package types

import (
	"net"
)

type LoadBalancerType string

const (
	RoundRobin LoadBalancerType = "RoundRobin"
	Random     LoadBalancerType = "Random"
)

type LoadBalancer interface {
	ChooseHost(context LoadBalancerContext) Host
}

type LoadBalancerContext interface {
	// compute an optional hash key to use during load balancing
	ComputeHashKey() HashedValue

	// get metadata match criteria used for selecting a subset of hosts
	MetadataMatchCriteria() MetadataMatchCriteria

	DownstreamConnection() net.Conn

	DownstreamHeaders() map[string]string
}

// SubSetLoadBalancer
type SubSetLoadBalancer interface {
	LoadBalancer
	// Find a host from the subsets, context is LoadBalancerContext
	// host_chosen = false and returned host = nil if no metadata found in context, if there is no matching subset
	// or if the matching subset contains no hosts/or no unhealthy hosts
	TryChooseHostFromContext(context LoadBalancerContext, hostChosen *bool) Host

	// used to created and update fallback subset
	UpdateFallbackSubset(priority uint32, hostAdded []Host, hostsRemoved []Host)

	HostMatches(kvs SubsetMetadata, host Host) bool

	// Iterates over the given metadata match criteria (which must be lexically sorted by key) and find
	// a matching LbSubsetEnryPtr, if any.
	FindSubset(matches []MetadataMatchCriterion) LBSubsetEntry

	// Given a vector of key-values (from extractSubsetMetadata), recursively finds the matching
	// LbSubsetEntryPtr.
	FindOrCreateSubset(subsets LbSubsetMap, kvs SubsetMetadata, idx uint32) LBSubsetEntry

	// Iterates over subset_keys looking up values from the given host's metadata. Each key-value pair
	// is appended to kvs. Returns a non-empty value if the host has a value for each key.
	ExtractSubsetMetadata(subsetKeys []string, host Host) SubsetMetadata

	// update or create subsets for one priority level
	Update(priority uint32, hostAdded []Host, hostsRemoved []Host)

	// ProcessSubsets called when host added or removed, to:
	// update lbsubset entry by updateCB or
	// create lbsubset entry by newCB
	ProcessSubsets(hostAdded []Host, hostsRemoved []Host,
		updateCB func(LBSubsetEntry), newCB func(LBSubsetEntry, HostPredicate, SubsetMetadata, bool))
}

// Entry stored in the subset hierarchy.
// children point to next lb subset map
// PrioritySubset point to the stored subset matched
type LBSubsetEntry interface {
	Initialized() bool

	Active() bool

	PrioritySubset() PrioritySubset
	
	SetPrioritySubset(PrioritySubset)

	Children() LbSubsetMap
}

// Represents a subset of an original HostSet.
type HostSubset interface {
	
	UpdateHostSubset(hostsAdded []Host, hostsRemoved []Host, predicate HostPredicate)

	//	TriggerCallbacks()   todo
	Empty() bool
	
	Hosts() []Host
}

// Represents a subset of an original PrioritySet.
type PrioritySubset interface {
	//PrioritySet
	
	Update(priority uint32, hostsAdded []Host, hostsRemoved []Host)
	
	Empty() bool
	
	GetOrCreateHostSubset(priority uint32) HostSubset
	
	TriggerCallbacks()
	
	CreateHostSet(priority uint32) HostSet
	
	LB() LoadBalancer
}

type FallBackPolicy uint8

const (
	NoFallBack FallBackPolicy = iota
	AnyEndPoint
	DefaultSubsetDefaultSubset
)

// Forms a trie-like structure. Route Metadata requires lexically sorted
// act as the root
type LbSubsetMap map[string]ValueSubsetMap

type ValueSubsetMap map[HashedValue]LBSubsetEntry

//type LBSubsetEntry struct {
//	children       types.LbSubsetMap
//	prioritySubset types.PrioritySubset
//}

type SubsetMetadata []Pair

type Pair struct {
	T1 string
	T2 HashedValue
}

type ResourcePriority uint8

const (
	Default ResourcePriority = 0
	High    ResourcePriority = 1
)
