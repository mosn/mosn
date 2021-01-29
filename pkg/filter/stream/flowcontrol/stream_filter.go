package flowcontrol

import (
	"context"
	"net"
	"strconv"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// FlowControlFilterName is the flow control stream filter name.
const (
	FlowControlFilterName = "flowControlFilter"
)

// defines the failed flags and code.
var (
	errorResponseFlag = []api.ResponseFlag{
		api.NoHealthyUpstream,
		api.UpstreamRequestTimeout,
		api.UpstreamLocalReset,
		api.UpstreamRemoteReset,
		api.UpstreamConnectionFailure,
		api.UpstreamConnectionTermination,
		api.NoRouteFound,
		api.DelayInjected,
		api.RateLimited,
		api.ReqEntityTooLarge,
	}

	errorResponseCode = []int{
		types.CodecExceptionCode,
		types.DeserialExceptionCode,
		types.PermissionDeniedCode,
		types.RouterUnavailableCode,
		types.NoHealthUpstreamCode,
		types.UpstreamOverFlowCode,
		types.TimeoutExceptionCode,
		types.LimitExceededCode,
	}
)

// defines the Resource context keys.
const (
	KeyInvokeSuccess = "success"
	KeySourceIp      = "X-CALLER-IP"
)

// StreamFilter represents the flow control stream filter.
type StreamFilter struct {
	Entry           *base.SentinelEntry
	BlockError      *base.BlockError
	Callbacks       Callbacks
	ReceiverHandler api.StreamReceiverFilterHandler
	SenderHandler   api.StreamSenderFilterHandler
	trafficType     base.TrafficType
	ranComplete     bool
}

// NewStreamFilter creates flow control filter.
func NewStreamFilter(callbacks Callbacks, trafficType base.TrafficType) *StreamFilter {
	callbacks.Init()
	return &StreamFilter{Callbacks: callbacks, trafficType: trafficType}
}

func (f *StreamFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.ReceiverHandler = handler
}

func (f *StreamFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	f.SenderHandler = handler
}

// OnReceive creates Resource and judges whether current request should be blocked.
func (f *StreamFilter) OnReceive(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if !f.Callbacks.Enabled() || f.Callbacks.ShouldIgnore(f, ctx, headers, buf, trailers) {
		return api.StreamFilterContinue
	}
	remoteAddr := f.ReceiverHandler.Connection().RemoteAddr()
	addr, _ := net.ResolveTCPAddr(remoteAddr.Network(), remoteAddr.String())
	if nil != addr {
		ctx = context.WithValue(ctx, KeySourceIp, addr.IP.String())
	}
	pr := f.Callbacks.ParseResource(ctx, headers, buf, trailers, f.trafficType)
	if pr == nil {
		log.DefaultLogger.Warnf("can't get Resource: %+v", headers)
		return api.StreamFilterContinue
	}

	entry, err := sentinel.Entry(pr.Resource.Name(), pr.Opts...)
	f.Entry = entry
	f.BlockError = err
	if err != nil {
		f.Callbacks.AfterBlock(f, ctx, headers, buf, trailers)
		return api.StreamFilterStop
	}

	f.Callbacks.AfterPass(f, ctx, headers, buf, trailers)
	return api.StreamFilterContinue
}

// Append marks whether the request is successful, do nothing if request was
// blocked in OnReceive phase.
func (f *StreamFilter) Append(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap) api.StreamFilterStatus {
	if f.BlockError != nil {
		return api.StreamFilterStop
	}

	if f.Entry == nil {
		return api.StreamFilterContinue
	}

	fail := f.isFail(ctx, headers, buf, trailers)
	if f.Entry.Context().Data == nil {
		f.Entry.Context().Data = make(map[interface{}]interface{})
	}
	f.Entry.Context().Data[KeyInvokeSuccess] = !fail
	return api.StreamFilterContinue
}

// OnDestroy calls the exit tasks.
func (f *StreamFilter) OnDestroy() {
	if f.Entry != nil && !f.ranComplete {
		f.ranComplete = true
		f.Callbacks.Exit(f)
		f.Entry.Exit()
	}
}

// isFail judges whether the request is failed according to response flags,
// response code and header status.
func (f *StreamFilter) isFail(ctx context.Context, headers types.HeaderMap,
	buf types.IoBuffer, trailers types.HeaderMap) bool {
	requestInfo := f.SenderHandler.RequestInfo()
	for _, responseFlag := range errorResponseFlag {
		if requestInfo.GetResponseFlag(responseFlag) {
			return true
		}
	}

	responseCode := f.SenderHandler.RequestInfo().ResponseCode()
	for _, errCode := range errorResponseCode {
		if responseCode == errCode {
			return true
		}
	}

	if code, ok := headers.Get(types.VarHeaderStatus); ok {
		if codeInt, err := strconv.Atoi(code); err == nil {
			for _, errCode := range errorResponseCode {
				if codeInt == errCode {
					return true
				}
			}
		}
	}

	if f.Callbacks != nil {
		return f.Callbacks.IsInvocationFail(ctx, headers, buf, trailers)
	}

	return false
}
