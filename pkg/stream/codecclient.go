package stream

import (
	"container/list"
	"context"
	"sync"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

// stream.CodecClient
// types.ReadFilter
// types.StreamConnectionEventListener
type codecClient struct {
	context context.Context

	Protocol   types.Protocol
	Connection types.ClientConnection
	Host       types.HostInfo
	Codec      types.ClientStreamConnection

	ActiveRequests *list.List
	AcrMux         sync.RWMutex

	CodecCallbacks            types.StreamConnectionEventListener
	CodecClientCallbacks      CodecClientCallbacks
	StreamConnectionCallbacks types.StreamConnectionEventListener
	ConnectedFlag             bool
	RemoteCloseFlag           bool
}

func NewCodecClient(context context.Context, prot types.Protocol, connection types.ClientConnection, host types.HostInfo) CodecClient {
	codecClient := &codecClient{
		context:        context,
		Protocol:       prot,
		Connection:     connection,
		Host:           host,
		ActiveRequests: list.New(),
	}

	if factory, ok := streamFactories[prot]; ok {
		codecClient.Codec = factory.CreateClientStream(context, connection, codecClient, codecClient)
	} else {
		return nil
	}

	connection.AddConnectionEventListener(codecClient)
	connection.FilterManager().AddReadFilter(codecClient)
	connection.SetNoDelay(true)

	return codecClient
}

func NewBiDirectCodeClient(context context.Context, prot types.Protocol, connection types.ClientConnection, host types.HostInfo,
	serverCallbacks types.ServerStreamConnectionEventListener) CodecClient {
	codecClient := &codecClient{
		Protocol:       prot,
		Connection:     connection,
		Host:           host,
		ActiveRequests: list.New(),
	}

	if factory, ok := streamFactories[prot]; ok {
		codecClient.Codec = factory.CreateBiDirectStream(context, connection, codecClient, serverCallbacks)
	} else {
		return nil
	}

	connection.AddConnectionEventListener(codecClient)
	connection.FilterManager().AddReadFilter(codecClient)
	connection.SetNoDelay(true)

	return codecClient
}

func (c *codecClient) Id() uint64 {
	return c.Connection.Id()
}

func (c *codecClient) AddConnectionCallbacks(cb types.ConnectionEventListener) {
	c.Connection.AddConnectionEventListener(cb)
}

func (c *codecClient) ActiveRequestsNum() int {
	c.AcrMux.RLock()
	defer c.AcrMux.RUnlock()

	return c.ActiveRequests.Len()
}

func (c *codecClient) SetConnectionStats(stats *types.ConnectionStats) {
	c.Connection.SetStats(stats)
}

func (c *codecClient) SetCodecClientCallbacks(cb CodecClientCallbacks) {
	c.CodecClientCallbacks = cb
}

func (c *codecClient) SetCodecConnectionCallbacks(cb types.StreamConnectionEventListener) {
	c.StreamConnectionCallbacks = cb
}

func (c *codecClient) RemoteClose() bool {
	return c.RemoteCloseFlag
}

func (c *codecClient) NewStream(streamId string, respDecoder types.StreamDecoder) types.StreamEncoder {
	ar := newActiveRequest(c, respDecoder)
	ar.requestEncoder = c.Codec.NewStream(streamId, ar)
	ar.requestEncoder.GetStream().AddEventListener(ar)

	c.AcrMux.Lock()
	ar.element = c.ActiveRequests.PushBack(ar)
	c.AcrMux.Unlock()

	return ar.requestEncoder
}

func (c *codecClient) Close() {
	c.Connection.Close(types.NoFlush, types.LocalClose)
}

// types.StreamConnectionEventListener
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
		c.AcrMux.RLock()
		defer c.AcrMux.RUnlock()

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
	c.Codec.OnUnderlyingConnectionAboveWriteBufferHighWatermark()
}

func (c *codecClient) OnBelowWriteBufferLowWatermark() {
	c.Codec.OnUnderlyingConnectionBelowWriteBufferLowWatermark()
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
	request.requestEncoder.GetStream().RemoveEventListener(request)
}

func (c *codecClient) deleteRequest(request *activeRequest) {
	c.AcrMux.Lock()
	defer c.AcrMux.Unlock()

	c.ActiveRequests.Remove(request.element)

	if c.CodecClientCallbacks != nil {
		c.CodecClientCallbacks.OnStreamDestroy()
	}
}

// types.StreamEventListener
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

func (r *activeRequest) OnAboveWriteBufferHighWatermark() {}

func (r *activeRequest) OnBelowWriteBufferLowWatermark() {}

func (r *activeRequest) OnDecodeHeaders(headers map[string]string, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseDecoder.OnDecodeHeaders(headers, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) OnDecodeData(data types.IoBuffer, endStream bool) {
	if endStream {
		r.onPreDecodeComplete()
	}

	r.responseDecoder.OnDecodeData(data, endStream)

	if endStream {
		r.onDecodeComplete()
	}
}

func (r *activeRequest) OnDecodeTrailers(trailers map[string]string) {
	r.onPreDecodeComplete()
	r.responseDecoder.OnDecodeTrailers(trailers)
	r.onDecodeComplete()
}

func (r *activeRequest) onPreDecodeComplete() {
	r.codecClient.responseDecodeComplete(r)
}

func (r *activeRequest) onDecodeComplete() {}
