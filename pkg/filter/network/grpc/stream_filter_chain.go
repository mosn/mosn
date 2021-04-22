package grpc

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
)

type grpcStreamFilterChain struct {
	*streamfilter.DefaultStreamFilterChainImpl
	phase types.Phase
	err error
}

func (sfc *grpcStreamFilterChain) AddStreamSenderFilter(filter api.StreamSenderFilter, phase api.SenderFilterPhase) {
	handler := newStreamSenderFilterHandler(sfc)
	filter.SetSenderFilterHandler(handler)
	sfc.DefaultStreamFilterChainImpl.AddStreamSenderFilter(filter, phase)
}

func (sfc *grpcStreamFilterChain) AddStreamReceiverFilter(filter api.StreamReceiverFilter, phase api.ReceiverFilterPhase) {
	handler := newStreamReceiverFilterHandler(sfc)
	filter.SetReceiveFilterHandler(handler)
	sfc.DefaultStreamFilterChainImpl.AddStreamReceiverFilter(filter, phase)
}

func (sfc *grpcStreamFilterChain) AddStreamAccessLog(accessLog api.AccessLog) {
	sfc.DefaultStreamFilterChainImpl.AddStreamAccessLog(accessLog)
}

// the stream filter chain are not allowed to be used anymore after calling this func
func (sfc *grpcStreamFilterChain) destroy() {
	// filter destroy
	sfc.DefaultStreamFilterChainImpl.OnDestroy()

	// reset fields
	streamfilter.PutStreamFilterChain(sfc.DefaultStreamFilterChainImpl)
	sfc.DefaultStreamFilterChainImpl = nil
}

func (sfc *grpcStreamFilterChain) receiverFilterStatusHandler(phase api.ReceiverFilterPhase, status api.StreamFilterStatus) {

}

func (sfc *grpcStreamFilterChain) senderFilterStatusHandler(phase api.SenderFilterPhase, status api.StreamFilterStatus) {

}


