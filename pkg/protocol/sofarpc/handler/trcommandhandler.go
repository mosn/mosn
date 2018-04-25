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

func (h *TrCommandHandler) HandleCommand(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(sofarpc.ProtoBasicCmd); ok {
		cmdCode := cmd.GetCmdCode()
		logger := log.ByContext(context)

		if processor, ok := h.processors[cmdCode]; ok {
			logger.Debugf("handle tr command")

			processor.Process(context, cmd, filter)
		} else {
			logger.Debugf("Unknown cmd code: [%x] while handle in TrCommandHandler.", cmdCode)
		}
	}
}

func (h *TrCommandHandler) RegisterProcessor(cmdCode int16, processor *sofarpc.RemotingProcessor) {
	if _, exists := h.processors[cmdCode]; exists {
		log.DefaultLogger.Warnf("tr cmd handler [%x] alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}
