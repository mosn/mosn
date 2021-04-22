package grpc

import (
	"context"
	"mosn.io/api"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
	"testing"
)

type mockFilter struct {}

func (m mockFilter) OnDestroy() {}

func (m mockFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	return api.StreamFilterContinue
}

func (m mockFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {}

func TestStreamFilter(t *testing.T)  {
	sfc := streamfilter.GetDefaultStreamFilterChain()
	ss := &grpcStreamFilterChain{
		DefaultStreamFilterChainImpl: sfc,
		phase:                        types.InitPhase,
		err:                          nil,
	}
	defer ss.destroy()
	ss.AddStreamReceiverFilter(&mockFilter{}, api.AfterRoute)
	status := ss.RunReceiverFilter(context.TODO(), api.AfterRoute, nil, nil, nil, ss.receiverFilterStatusHandler)
	if status != api.StreamFilterContinue {
		t.Fatalf("TestStreamFilter status: %v not equals with %v", status, api.StreamFilterContinue)
	}
}

