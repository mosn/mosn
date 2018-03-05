package handler

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"fmt"
)

type BoltCommandHandler struct {
	processors map[int16]protocol.RemotingProcessor
}

func NewBoltCommandHandler() *BoltCommandHandler{
	return &BoltCommandHandler{
		processors:map[int16]protocol.RemotingProcessor{
			protocol.RPC_REQUEST: &BoltRequestProcessor{},
			protocol.RPC_RESPONSE: &BoltResponseProcessor{},
			protocol.HEARTBEAT:&BoltHbProcessor{},
		},
	}
}

func (h *BoltCommandHandler) HandleCommand(ctx interface{}, msg interface{}){
	if cmd, ok := msg.(protocol.RpcCommand); ok {
		cmdCode := cmd.GetCmdCode()
		if processor, ok := h.processors[cmdCode]; ok{
			fmt.Println("handle command")
			processor.Process(ctx, cmd,nil)
		}else{
			fmt.Println("Unknown cmd code: [", cmdCode, "] while handle in BoltCommandHandler.")
		}
	}
}

func (h *BoltCommandHandler) RegisterProcessor(cmdCode int16, processor *protocol.RemotingProcessor){
	if _, exists := h.processors[cmdCode]; exists {
		fmt.Println("handler alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}