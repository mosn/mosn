package handler

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/codec"
	"fmt"
)

type TrCommandHandler struct {
	processors map[int16]protocol.RemotingProcessor
}

func NewTrCommandHandler() *TrCommandHandler{
	return &TrCommandHandler{
		processors:map[int16]protocol.RemotingProcessor{
			protocol.RPC_REQUEST: &TrRequestProcessor{},
			protocol.RPC_RESPONSE: &TrResponseProcessor{},
			protocol.HEARTBEAT:&TrHbProcessor{},
		},
	}
}

func (h *TrCommandHandler) HandleCommand(ctx interface{}, msg interface{}){
	if cmd, ok := msg.(*codec.BoltCommand); ok {
		cmdCode := cmd.GetCmdCode()
		if processor, ok := h.processors[cmdCode]; ok{
			processor.Process(ctx, cmd,nil)
		}else{
			fmt.Println("Unknown cmd code: [", cmdCode, "] while handle in TrCommandHandler.")
		}
	}
}

func (h *TrCommandHandler) RegisterProcessor(cmdCode int16, processor *protocol.RemotingProcessor){
	if _, exists := h.processors[cmdCode]; exists {
		fmt.Println("handler alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}