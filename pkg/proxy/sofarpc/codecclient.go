package sofarpc

import (
	"container/list"
	"gitlab.alipay-inc.com/afe/mosn/pkg/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// proxy.CodecClient
// types.ReadFilter
// SOFA-RPC client
type codecClient struct {
	proxy.BaseCodeClient
}

func newCodecClient(protocol types.Protocol, connection types.ClientConnection, host types.HostInfo) proxy.CodecClient {
	codecClient := &codecClient{
		proxy.BaseCodeClient{
			Protocol:       protocol,
			Connection:     connection,
			Host:           host,
			ActiveRequests: list.New(),
		},
	}
	codecClient.Codec = newClientStreamConnection(connection, codecClient)
	connection.FilterManager().AddReadFilter(codecClient)

	return codecClient
}

func (c *codecClient) NewStream(streamId uint32, respDecoder types.StreamDecoder) types.StreamEncoder {
	ar := newActiveRequest(c, respDecoder)
	ar.requestEncoder = c.Codec.NewStream(streamId, ar)
	ar.requestEncoder.GetStream().AddCallbacks(ar)

	ele := c.ActiveRequests.PushBack(ar)
	ar.element = ele

	return ar.requestEncoder
}

func (c *codecClient) Close() {
	c.Connection.Close(types.NoFlush, types.LocalClose)
}

func (c *codecClient) OnGoAway() {
	c.CodecCallbacks.OnGoAway()
}

// conn callbacks
func (c *codecClient) OnEvent(event types.ConnectionEvent) {
	switch event {
	case types.Connected:
		c.ConnectedFlag = true
	case types.RemoteClose:
		c.RemoteCloseFlag = true
	}

	if event.IsClose() {
		for ar := c.ActiveRequests.Front(); ar != nil; ar = ar.Next() {
			reason := types.StreamConnectionFailed
			if c.ConnectedFlag {
				reason = types.StreamConnectionTermination
			}

			ar.Value.(*activeRequest).requestEncoder.GetStream().ResetStream(reason)
		}
	}
}

func (c *codecClient) OnAboveWriteBufferHighWatermark() {
	// todo
}

func (c *codecClient) OnBelowWriteBufferLowWatermark() {
	// todo
}

// read filter, recv upstream data
func (c *codecClient) OnData(buffer types.IoBuffer) types.FilterStatus {
	c.Codec.Dispatch(buffer)

	return types.StopIteration
}

func (c *codecClient) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (c *codecClient) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}

func (c *codecClient) onReset(request *activeRequest, reason types.StreamResetReason) {
	if c.CodecClientCallbacks != nil {
		c.CodecClientCallbacks.OnStreamReset(reason)
	}

	c.deleteRequest(request)
}

func (c *codecClient) responseDecodeComplete(request *activeRequest) {
	c.deleteRequest(request)

	request.requestEncoder.GetStream().RemoveCallbacks(request)
}

func (c *codecClient) deleteRequest(request *activeRequest) {
	c.ActiveRequests.Remove(request.element)

	if c.CodecClientCallbacks != nil {
		c.CodecClientCallbacks.OnStreamDestroy()
	}
}

// types.StreamCallbacks
// types.StreamDecoderWrapper
type activeRequest struct {
	codecClient     *codecClient
	responseDecoder types.StreamDecoder
	requestEncoder  types.StreamEncoder
	element         *list.Element
}

func newActiveRequest(codecClient *codecClient, streamDecoder types.StreamDecoder) *activeRequest {
	return &activeRequest{
		codecClient:     codecClient,
		responseDecoder: streamDecoder,
	}
}

func (r *activeRequest) OnResetStream(reason types.StreamResetReason) {
	r.codecClient.onReset(r, reason)
}

func (r *activeRequest) OnAboveWriteBufferHighWatermark() {
	// todo
}

func (r *activeRequest) OnBelowWriteBufferLowWatermark() {
	// todo
}

func (r *activeRequest) DecodeHeaders(headers map[string]string, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseDecoder.DecodeHeaders(headers, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) DecodeData(data types.IoBuffer, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseDecoder.DecodeData(data, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) DecodeTrailers(trailers map[string]string) {
	r.onPreDecodeComplete()
	r.responseDecoder.DecodeTrailers(trailers)
	r.onDecodeComplete()
}

func (r *activeRequest) DecodeComplete(data types.IoBuffer) {
	r.responseDecoder.DecodeComplete(data)
}

func (r *activeRequest) onPreDecodeComplete() {
	r.codecClient.responseDecodeComplete(r)
}

func (r *activeRequest) onDecodeComplete() {}
