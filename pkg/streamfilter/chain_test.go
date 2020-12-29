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
