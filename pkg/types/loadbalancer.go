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
	"context"
	"net"

	"mosn.io/api"
)

// LoadBalancerType is the load balancer's type
type LoadBalancerType string

// The load balancer's types
const (
	RoundRobin         LoadBalancerType = "LB_ROUNDROBIN"
	Random             LoadBalancerType = "LB_RANDOM"
	WeightedRoundRobin LoadBalancerType = "LB_WEIGHTED_ROUNDROBIN"
	ORIGINAL_DST       LoadBalancerType = "LB_ORIGINAL_DST"
	LeastActiveRequest LoadBalancerType = "LB_LEAST_REQUEST"
	Maglev             LoadBalancerType = "LB_MAGLEV"
)

// LoadBalancer is a upstream load balancer.
// When a request comes, the LoadBalancer will choose a upstream cluster's host to handle the request.
type LoadBalancer interface {
	// ChooseHost chooses a host based on the load balancer context
	ChooseHost(context LoadBalancerContext) Host
	// IsExistsHosts checks the load balancer contains hosts or not
	// It will not be effect the load balancer's index
	IsExistsHosts(api.MetadataMatchCriteria) bool

	HostNum(api.MetadataMatchCriteria) int
}

// LoadBalancerContext contains the information for choose a host
type LoadBalancerContext interface {

	// MetadataMatchCriteria gets metadata match criteria used for selecting a subset of hosts
	MetadataMatchCriteria() api.MetadataMatchCriteria

	// DownstreamConnection returns the downstream connection.
	DownstreamConnection() net.Conn

	// DownstreamHeaders returns the downstream headers map.
	DownstreamHeaders() api.HeaderMap

	// DownstreamContext returns the downstream context
	DownstreamContext() context.Context

	// Downstream cluster info
	DownstreamCluster() ClusterInfo

	// Downstream route info
	DownstreamRoute() api.Route
}

// LBSubsetEntry is a entry that stored in the subset hierarchy.
type LBSubsetEntry interface {
	// Initialized returns the entry is initialized or not.
	Initialized() bool

	// Active returns the entry is active or not.
	Active() bool

	// Children returns the next lb subset map
	Children() LbSubsetMap

	CreateLoadBalancer(ClusterInfo, HostSet)

	LoadBalancer() LoadBalancer

	HostNum() int
}

// FallBackPolicy type
type FallBackPolicy uint8

// FallBackPolicy types
const (
	NoFallBack FallBackPolicy = iota
	AnyEndPoint
	DefaultSubset
)

// LbSubsetMap is a trie-like structure. Route Metadata requires lexically sorted
// act as the root.
type LbSubsetMap map[string]ValueSubsetMap

// ValueSubsetMap is a LBSubsetEntry map.
type ValueSubsetMap map[string]LBSubsetEntry

//type LBSubsetEntry struct {
//	children       types.LbSubsetMap
//	prioritySubset types.PrioritySubset
//}

// SubsetMetadata is a vector of key-values
type SubsetMetadata []Pair

// Pair is a key-value pair that contains metadata.
type Pair struct {
	T1 string
	T2 string
}
