package proxy

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strconv"
	"math/rand"
	"time"
)

type retrystate struct {
	retryPolicy     types.RetryPolicy
	requestHeaders  map[string]string
	cluster         types.ClusterInfo
	retryOn         bool
	retiesRemaining int
}

func newRetryState(retryPolicy types.RetryPolicy, requestHeaders map[string]string, cluster types.ClusterInfo) *retrystate {
	rs := &retrystate{
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

func (r *retrystate) shouldRetry(headers map[string]string, reason types.StreamResetReason) bool {
	if r.retiesRemaining == 0 {
		return false
	}

	r.retiesRemaining--

	if !r.doCheckRetry(headers, reason) {
		return false
	}

	return true
}

func (r *retrystate) scheduleRetry(doRetry func()) {
	// todo: better alth
	timeout := rand.Intn(10)

	go func() {
		select {
		case <-time.After(time.Duration(timeout) * time.Second):
			doRetry()
		}
	}()
}

func (r *retrystate) doCheckRetry(headers map[string]string, reason types.StreamResetReason) bool {
	if reason != "" && reason == types.StreamOverflow {
		return false
	}

	if r.retryOn {
		if code, ok := headers["Status"]; ok {
			codeValue, _ := strconv.Atoi(code)

			return codeValue >= 500
		}

		// todo: more conditions
	}

	return false
}
