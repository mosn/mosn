package filter

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

const UndefinedFilterPhase api.FilterPhase = 99999

type StreamFilterChainStatus int

const (
	// continue running filters
	StreamFilterChainContinue StreamFilterChainStatus = 0
	// stop running filters, next time should retry the current filter
	StreamFilterChainStop StreamFilterChainStatus = 1
	// stop running filters and reset index, next time should run the first filter
	StreamFilterChainReset StreamFilterChainStatus = 2
)

type StreamFilterStatusHandler func(status api.StreamFilterStatus) StreamFilterChainStatus

func DefaultStreamFilterStatusHandler(status api.StreamFilterStatus) StreamFilterChainStatus {
	switch status {
	case api.StreamFilterContinue:
		return StreamFilterChainContinue
	case api.StreamFilterStop:
		return StreamFilterChainReset
	case api.StreamFiltertermination:
		return StreamFilterChainReset
	}
	return StreamFilterChainContinue
}

type StreamFilterManager interface {
	api.StreamFilterChainFactoryCallbacks

	RunReceiverFilter(ctx context.Context, phase api.FilterPhase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap, statusHandler StreamFilterStatusHandler) api.StreamFilterStatus

	RunSenderFilter(ctx context.Context, phase api.FilterPhase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap, statusHandler StreamFilterStatusHandler) api.StreamFilterStatus

	Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo)

	OnDestroy()
}

type StreamReceiverFilterWithPhase interface {
	api.StreamReceiverFilter
	CheckPhase(phase api.FilterPhase) bool
}

type StreamReceiverFilterWithPhaseImpl struct {
	filter api.StreamReceiverFilter
	phase  api.FilterPhase
}

func NewStreamReceiverFilterWithPhaseImpl(f api.StreamReceiverFilter, p api.FilterPhase) *StreamReceiverFilterWithPhaseImpl {
	return &StreamReceiverFilterWithPhaseImpl{
		filter: f,
		phase:  p,
	}
}

func (s *StreamReceiverFilterWithPhaseImpl) OnDestroy() {
	s.filter.OnDestroy()
}

func (s *StreamReceiverFilterWithPhaseImpl) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return s.filter.OnReceive(ctx, headers, buf, trailers)
}

func (s *StreamReceiverFilterWithPhaseImpl) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	s.SetReceiveFilterHandler(handler)
}

func (s *StreamReceiverFilterWithPhaseImpl) CheckPhase(phase api.FilterPhase) bool {
	return s.phase == phase
}

type StreamSenderFilterWithPhase interface {
	api.StreamSenderFilter
	CheckPhase(phase api.FilterPhase) bool
}

type StreamSenderFilterWithPhaseImpl struct {
	filter api.StreamSenderFilter
	phase  api.FilterPhase
}

func NewStreamSenderFilterWithPhaseImpl(f api.StreamSenderFilter, p api.FilterPhase) *StreamSenderFilterWithPhaseImpl {
	return &StreamSenderFilterWithPhaseImpl{
		filter: f,
		phase:  p,
	}
}

func (s *StreamSenderFilterWithPhaseImpl) OnDestroy() {
	s.filter.OnDestroy()
}

func (s *StreamSenderFilterWithPhaseImpl) Append(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return s.filter.Append(ctx, headers, buf, trailers)
}

func (s *StreamSenderFilterWithPhaseImpl) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	s.filter.SetSenderFilterHandler(handler)
}

func (s *StreamSenderFilterWithPhaseImpl) CheckPhase(phase api.FilterPhase) bool {
	return true
}

type DefaultStreamFilterManagerImpl struct {
	senderFilters      []StreamSenderFilterWithPhase
	senderFiltersIndex int

	receiverFilters      []StreamReceiverFilterWithPhase
	receiverFiltersIndex int

	streamAccessLogs []api.AccessLog
}

func (d *DefaultStreamFilterManagerImpl) AddStreamSenderFilter(filter api.StreamSenderFilter) {
	f := NewStreamSenderFilterWithPhaseImpl(filter, UndefinedFilterPhase)
	d.senderFilters = append(d.senderFilters, f)
}

func (d *DefaultStreamFilterManagerImpl) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.FilterPhase) {
	f := NewStreamReceiverFilterWithPhaseImpl(filter, p)
	d.receiverFilters = append(d.receiverFilters, f)
}

func (d *DefaultStreamFilterManagerImpl) AddStreamAccessLog(accessLog api.AccessLog) {
	d.streamAccessLogs = append(d.streamAccessLogs, accessLog)
}

func (d *DefaultStreamFilterManagerImpl) RunReceiverFilter(ctx context.Context, phase api.FilterPhase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler StreamFilterStatusHandler) (filterStatus api.StreamFilterStatus) {

	if statusHandler == nil {
		statusHandler = DefaultStreamFilterStatusHandler
	}
	filterStatus = api.StreamFilterContinue

	for ; d.receiverFiltersIndex < len(d.receiverFilters); d.receiverFiltersIndex++ {
		filter := d.receiverFilters[d.receiverFiltersIndex]
		if !filter.CheckPhase(phase) {
			continue
		}

		filterStatus = filter.OnReceive(ctx, headers, data, trailers)

		chainStatus := statusHandler(filterStatus)
		switch chainStatus {
		case StreamFilterChainContinue:
			continue
		case StreamFilterChainStop:
			return
		case StreamFilterChainReset:
			d.receiverFiltersIndex = 0
			return
		default:
			continue
		}
	}
	d.receiverFiltersIndex = 0
	return
}

func (d *DefaultStreamFilterManagerImpl) RunSenderFilter(ctx context.Context, phase api.FilterPhase, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap,
	statusHandler StreamFilterStatusHandler) (filterStatus api.StreamFilterStatus) {

	if statusHandler == nil {
		statusHandler = DefaultStreamFilterStatusHandler
	}
	filterStatus = api.StreamFilterContinue

	for ; d.senderFiltersIndex < len(d.senderFilters); d.senderFiltersIndex++ {
		filter := d.senderFilters[d.senderFiltersIndex]
		if !filter.CheckPhase(phase) {
			continue
		}

		filterStatus = filter.Append(ctx, headers, data, trailers)

		chainStatus := statusHandler(filterStatus)
		switch chainStatus {
		case StreamFilterChainContinue:
			continue
		case StreamFilterChainStop:
			return
		case StreamFilterChainReset:
			d.receiverFiltersIndex = 0
			return
		default:
			continue
		}
	}
	d.senderFiltersIndex = 0
	return
}

func (d *DefaultStreamFilterManagerImpl) Log(ctx context.Context, reqHeaders api.HeaderMap, respHeaders api.HeaderMap, requestInfo api.RequestInfo) {
	for _, l := range d.streamAccessLogs {
		l.Log(ctx, reqHeaders, respHeaders, requestInfo)
	}
}

func (d *DefaultStreamFilterManagerImpl) OnDestroy() {
	for _, filter := range d.receiverFilters {
		filter.OnDestroy()
	}
	for _, filter := range d.senderFilters {
		filter.OnDestroy()
	}
}
