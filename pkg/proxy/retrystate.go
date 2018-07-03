package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"math/rand"
	"strconv"
	"time"
)

type retryState struct {
	retryPolicy     types.RetryPolicy
	requestHeaders  map[string]string
	cluster         types.ClusterInfo
	retryOn         bool
	retiesRemaining uint32
}

func newRetryState(retryPolicy types.RetryPolicy,
	requestHeaders map[string]string, cluster types.ClusterInfo) *retryState {
	rs := &retryState{
		retryPolicy:     retryPolicy,
		requestHeaders:  requestHeaders,
		cluster:         cluster,
		retryOn:         retryPolicy.RetryOn(),
		retiesRemaining: 1,
	}

	if retryPolicy.NumRetries() > rs.retiesRemaining {
		rs.retiesRemaining = retryPolicy.NumRetries()
	}

	return rs
}

func (r *retryState) shouldRetry(headers map[string]string, reason types.StreamResetReason) bool {
	if r.retiesRemaining == 0 {
		return false
	}

	if !r.doCheckRetry(headers, reason) {
		return false
	}

	return true
}

func (r *retryState) scheduleRetry(doRetry func()) *timer {
	r.retiesRemaining--

	// todo: better alth
	timeout := rand.Intn(10)

	timer := newTimer(doRetry, time.Duration(timeout)*time.Second)
	timer.start()

	r.cluster.Stats().UpstreamRequestRetry.Inc(1)

	return timer
}

func (r *retryState) doCheckRetry(headers map[string]string, reason types.StreamResetReason) bool {
	if reason == types.StreamOverflow {
		return false
	}

	if r.retryOn {
		if code, ok := headers[types.HeaderStatus]; ok {
			codeValue, _ := strconv.Atoi(code)

			return codeValue >= 500
		}

		// todo: more conditions
	}

	return false
}
