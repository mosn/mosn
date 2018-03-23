package proxy

import (
	"time"
	"container/list"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.StreamCallbacks
// types.StreamDecoder
// types.PoolCallbacks
type upstreamRequest struct {
	proxy          *proxy
	element        *list.Element
	activeStream   *activeStream
	host           types.Host
	requestInfo    types.RequestInfo
	requestEncoder types.StreamEncoder
	connPool       types.ConnectionPool
}

func (r *upstreamRequest) resetStream() {
	if r.requestEncoder != nil {
		r.requestEncoder.GetStream().RemoveCallbacks(r)
		r.requestEncoder.GetStream().ResetStream(types.StreamLocalReset)
	}
}

// types.StreamCallbacks
func (r *upstreamRequest) OnResetStream(reason types.StreamResetReason) {
	r.requestEncoder = nil

	// todo: check if we get a reset on encode request headers. e.g. encode failed
	r.requestInfo.SetResponseFlag(r.proxy.streamResetReasonToResponseFlag(reason))
	r.activeStream.onUpstreamReset(UpstreamReset, reason)
}

func (r *upstreamRequest) OnAboveWriteBufferHighWatermark() {}

func (r *upstreamRequest) OnBelowWriteBufferLowWatermark() {}

// types.StreamDecoder
func (r *upstreamRequest) OnDecodeHeaders(headers map[string]string, endStream bool) {
	r.activeStream.onUpstreamHeaders(headers, endStream)
}

func (r *upstreamRequest) OnDecodeData(data types.IoBuffer, endStream bool) {
	r.activeStream.onUpstreamData(data, endStream)
}

func (r *upstreamRequest) OnDecodeTrailers(trailers map[string]string) {
	r.activeStream.onUpstreamTrailers(trailers)
}

func (r *upstreamRequest) OnDecodeComplete(data types.IoBuffer) {}

func (r *upstreamRequest) responseDecodeComplete() {}

// types.PoolCallbacks
func (r *upstreamRequest) OnPoolFailure(streamId uint32, reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.OnResetStream(resetReason)
}

func (r *upstreamRequest) OnPoolReady(streamId uint32, encoder types.StreamEncoder, host types.Host) {
	r.requestEncoder = encoder

	r.requestInfo.OnUpstreamHostSelected(host)
	r.requestEncoder.GetStream().AddCallbacks(r)

	// todo
	r.requestEncoder.EncodeHeaders(r.activeStream.downstreamHeaders, false)

	// todo: check if we get a reset on encode headers
}

func (r *upstreamRequest) setupPerTryTimeout() {
	timeout := r.activeStream.timeout

	if timeout.TryTimeout > 0 {
		go func() {
			// todo: support cancel
			select {
			case <-time.After(timeout.TryTimeout * time.Second):
				r.onPerTryTimeout()
			}
		}()
	}
}

func (r *upstreamRequest) onPerTryTimeout() {
	r.resetStream()

	r.requestInfo.SetResponseFlag(types.UpstreamRequestTimeout)
	r.activeStream.onUpstreamReset(UpstreamPerTryTimeout, types.StreamLocalReset)
}
