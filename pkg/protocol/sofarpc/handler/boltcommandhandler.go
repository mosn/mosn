package handler

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"fmt"
)

type BoltCommandHandler struct {
	processors map[int16]sofarpc.RemotingProcessor
}

func NewBoltCommandHandler() *BoltCommandHandler {
	return &BoltCommandHandler{
		processors: map[int16]sofarpc.RemotingProcessor{
			sofarpc.RPC_REQUEST:  &BoltRequestProcessor{},
			sofarpc.RPC_RESPONSE: &BoltResponseProcessor{},
			sofarpc.HEARTBEAT:    &BoltHbProcessor{},
		},
	}
}

func (h *BoltCommandHandler) HandleCommand(ctx interface{}, msg interface{}) {
	if cmd, ok := msg.(sofarpc.RpcCommand); ok {
		cmdCode := cmd.GetCmdCode()
		if processor, ok := h.processors[cmdCode]; ok {
			fmt.Println("handle command")
			processor.Process(ctx, cmd, nil)
		} else {
			fmt.Println("Unknown cmd code: [", cmdCode, "] while handle in BoltCommandHandler.")
		}
	}
}

func (h *BoltCommandHandler) RegisterProcessor(cmdCode int16, processor *sofarpc.RemotingProcessor) {
	if _, exists := h.processors[cmdCode]; exists {
		fmt.Println("handler alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}
