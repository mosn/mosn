package proxy

import (
	"container/list"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// types.StreamEventListener
// types.StreamDecoder
// types.PoolEventListener
type upstreamRequest struct {
	proxy          *proxy
	element        *list.Element
	activeStream   *activeStream
	host           types.Host
	requestEncoder types.StreamEncoder
	connPool       types.ConnectionPool

	// ~~~ upstream response buf
	upstreamRespHeaders map[string]string

	//~~~ state
	encodeComplete bool
	dataEncoded    bool
	trailerEncoded bool
}

func (r *upstreamRequest) resetStream() {
	if r.requestEncoder != nil {
		r.requestEncoder.GetStream().RemoveEventListener(r)
		r.requestEncoder.GetStream().ResetStream(types.StreamLocalReset)
	}
}

// types.StreamEventListener
func (r *upstreamRequest) OnResetStream(reason types.StreamResetReason) {
	r.requestEncoder = nil

	// todo: check if we get a reset on encode request headers. e.g. encode failed
	r.activeStream.onUpstreamReset(UpstreamReset, reason)
}

func (r *upstreamRequest) OnAboveWriteBufferHighWatermark() {
	r.activeStream.onUpstreamAboveWriteBufferHighWatermark()
}

func (r *upstreamRequest) OnBelowWriteBufferLowWatermark() {
	r.activeStream.onUpstreamBelowWriteBufferHighWatermark()
}

// types.StreamDecoder
// Method to decode upstream's response message
func (r *upstreamRequest) OnDecodeHeaders(headers map[string]string, endStream bool) {
	r.upstreamRespHeaders = headers
	r.activeStream.onUpstreamHeaders(headers, endStream)
}

func (r *upstreamRequest) OnDecodeData(data types.IoBuffer, endStream bool) {
	r.activeStream.onUpstreamData(data, endStream)
}

func (r *upstreamRequest) OnDecodeTrailers(trailers map[string]string) {
	r.activeStream.onUpstreamTrailers(trailers)
}

func (r *upstreamRequest) OnDecodeError(err error,headers map[string]string){
}

// ~~~ encode request wrapper

func (r *upstreamRequest) encodeHeaders(headers map[string]string, endStream bool) {
	r.encodeComplete = endStream
	streamID := ""

	if streamid, ok := headers[types.HeaderStreamID]; ok {
		streamID = streamid
	}

	r.connPool.NewStream(r.proxy.context, streamID, r, r)
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

// types.PoolEventListener
func (r *upstreamRequest) OnPoolFailure(streamId string, reason types.PoolFailureReason, host types.Host) {
	var resetReason types.StreamResetReason

	switch reason {
	case types.Overflow:
		resetReason = types.StreamOverflow
	case types.ConnectionFailure:
		resetReason = types.StreamConnectionFailed
	}

	r.OnResetStream(resetReason)
}

func (r *upstreamRequest) OnPoolReady(streamId string, encoder types.StreamEncoder, host types.Host) {
	r.requestEncoder = encoder
	r.requestEncoder.GetStream().AddEventListener(r)

	endStream := r.encodeComplete && !r.dataEncoded && !r.trailerEncoded
	r.requestEncoder.EncodeHeaders(r.activeStream.downstreamReqHeaders, endStream)

	r.activeStream.requestInfo.OnUpstreamHostSelected(host)
	r.activeStream.requestInfo.SetUpstreamLocalAddress(host.Address())

	// todo: check if we get a reset on encode headers
}
