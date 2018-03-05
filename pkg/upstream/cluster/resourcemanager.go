package cluster

import (
	"sync/atomic"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// ResourceManager
type resourcemanager struct {
	connections     *resource
	pendingRequests *resource
	requests        *resource
}

func NewResourceManager(maxConnections uint64, maxPendingRequests uint64, maxRequests uint64) types.ResourceManager {
	return &resourcemanager{
		connections: &resource{
			max: maxConnections,
		},
		pendingRequests: &resource{
			max: maxPendingRequests,
		},
		requests: &resource{
			max: maxRequests,
		},
	}
}

func (rm *resourcemanager) ConnectionResource() types.Resource {
	return rm.connections
}

func (rm *resourcemanager) PendingRequests() types.Resource {
	return rm.pendingRequests
}

func (rm *resourcemanager) Requests() types.Resource {
	return rm.requests
}

// Resource
type resource struct {
	current int64
	max     uint64
}

func (r *resource) CanCreate() bool {
	curValue := atomic.LoadInt64(&r.current)

	return uint64(curValue) < r.Max()
}

func (r *resource) Increase() {
	atomic.AddInt64(&r.current, 1)
}

func (r *resource) Decrease() {
	atomic.AddInt64(&r.current, -1)
}

func (r *resource) Max() uint64 {
	return r.max
}
