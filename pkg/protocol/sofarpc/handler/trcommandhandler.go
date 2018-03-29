package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
)

type TrCommandHandler struct {
	processors map[int16]sofarpc.RemotingProcessor
}

func NewTrCommandHandler() *TrCommandHandler {
	return &TrCommandHandler{
		processors: map[int16]sofarpc.RemotingProcessor{
			sofarpc.RPC_REQUEST:  &TrRequestProcessor{},
			sofarpc.RPC_RESPONSE: &TrResponseProcessor{},
			sofarpc.HEARTBEAT:    &TrHbProcessor{},
		},
	}
}

func (h *TrCommandHandler) HandleCommand(ctx interface{}, msg interface{}) {
	//if cmd, ok := msg.(sofarpc.TrRequestCommand); ok {
	//	cmdCode := cmd.GetCmdCode()
	//	if processor, ok := h.processors[cmdCode]; ok {
	//		processor.Process(ctx, cmd, nil)
	//	} else {
	//		fmt.Println("Unknown cmd code: [", cmdCode, "] while handle in TrCommandHandler.")
	//	}
	//}
}

func (h *TrCommandHandler) RegisterProcessor(cmdCode int16, processor *sofarpc.RemotingProcessor) {
	if _, exists := h.processors[cmdCode]; exists {
		fmt.Println("handler alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}
