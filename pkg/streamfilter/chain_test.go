package streamfilter

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/mock"
)

func TestStreamFilterChainPool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chain := GetDefaultStreamFilterChain()

	receiverFilter := mock.NewMockStreamReceiverFilter(ctrl)
	chain.AddStreamReceiverFilter(receiverFilter, api.BeforeRoute)

	senderFilter := mock.NewMockStreamSenderFilter(ctrl)
	chain.AddStreamSenderFilter(senderFilter, api.BeforeSend)

	PutStreamFilterChain(chain)

	chain2 := GetDefaultStreamFilterChain()

	assert.Equal(t, chain, chain2)

	assert.Equal(t, len(chain2.senderFilters), 0)
	assert.Equal(t, len(chain2.senderFiltersPhase), 0)
	assert.Equal(t, chain2.senderFiltersIndex, 0)
	assert.Equal(t, len(chain2.receiverFilters), 0)
	assert.Equal(t, len(chain2.receiverFiltersPhase), 0)
	assert.Equal(t, chain2.receiverFiltersIndex, 0)
}

func TestStreamFilterChainSetFilterHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chain := GetDefaultStreamFilterChain()

	setHandlerCount := 0

	for i := 0; i < 5; i++ {
		receiverFilter := mock.NewMockStreamReceiverFilter(ctrl)
		receiverFilter.EXPECT().SetReceiveFilterHandler(gomock.Any()).AnyTimes().Do(func(api.StreamReceiverFilterHandler) {
			setHandlerCount++
		})
		chain.AddStreamReceiverFilter(receiverFilter, api.BeforeRoute)

		senderFilter := mock.NewMockStreamSenderFilter(ctrl)
		senderFilter.EXPECT().SetSenderFilterHandler(gomock.Any()).AnyTimes().Do(func(api.StreamSenderFilterHandler) {
			setHandlerCount++
		})
		chain.AddStreamSenderFilter(senderFilter, api.BeforeSend)
	}

	chain.SetReceiveFilterHandler(nil)
	rFilterHandler := mock.NewMockStreamReceiverFilterHandler(ctrl)
	chain.SetReceiveFilterHandler(rFilterHandler)

	chain.SetSenderFilterHandler(nil)
	sFilterHandler := mock.NewMockStreamSenderFilterHandler(ctrl)
	chain.SetSenderFilterHandler(sFilterHandler)

	assert.Equal(t, setHandlerCount, 10)
}
