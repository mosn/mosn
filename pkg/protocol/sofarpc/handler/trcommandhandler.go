package handler

import (
	"context"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
)

type TrCommandHandler struct {
	processors map[int16]sofarpc.RemotingProcessor
}

func NewTrCommandHandler() *TrCommandHandler {
	return &TrCommandHandler{
		processors: map[int16]sofarpc.RemotingProcessor{
			sofarpc.TR_REQUEST:   &TrRequestProcessor{},
			sofarpc.TR_RESPONSE:  &TrResponseProcessor{},
			sofarpc.TR_HEARTBEAT: &TrHbProcessor{},
		},
	}
}

func (h *TrCommandHandler) HandleCommand(ctx interface{}, msg interface{}, context context.Context) {
	if cmd, ok := msg.(sofarpc.ProtoBasicCmd); ok {
		cmdCode := cmd.GetCmdCode()
		if processor, ok := h.processors[cmdCode]; ok {
			log.DefaultLogger.Debugf("handle command")

			processor.Process(ctx, cmd, nil, context)
		} else {
			log.DefaultLogger.Debugf("Unknown cmd code: [", cmdCode, "] while handle in TrCommandHandler.")
		}
	}
}

func (h *TrCommandHandler) RegisterProcessor(cmdCode int16, processor *sofarpc.RemotingProcessor) {
	if _, exists := h.processors[cmdCode]; exists {
		log.DefaultLogger.Debugf("handler alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}
