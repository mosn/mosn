package proxy

import (
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

	//~~~ state
	encodeComplete bool
	dataEncoded    bool
	trailerEncoded bool
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

// ~~~ encode request wrapper

func (r *upstreamRequest) encodeHeaders(headers map[string]string, endStream bool) {
	r.encodeComplete = endStream
	r.connPool.NewStream(0, r, r)    //调用STREAM层的NEW stream函数
}

func (r *upstreamRequest) encodeData(data types.IoBuffer, endStream bool) {
	r.encodeComplete = endStream
	r.dataEncoded = true
	r.requestEncoder.EncodeData(data, endStream)
}

func (r *upstreamRequest) encodeTrailers(trailers map[string]string) {
	r.encodeComplete = true
	r.trailerEncoded = true
	r.requestEncoder.EncodeTrailers(trailers)
}

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
	r.requestEncoder.GetStream().AddCallbacks(r)

	endStream := r.encodeComplete && !r.dataEncoded && !r.trailerEncoded
	r.requestEncoder.EncodeHeaders(r.activeStream.downstreamHeaders, endStream)

	r.requestInfo.OnUpstreamHostSelected(host)
	r.activeStream.requestInfo.OnUpstreamHostSelected(host)
	r.activeStream.requestInfo.SetUpstreamLocalAddress(host.Address())

	// todo: check if we get a reset on encode headers
}
