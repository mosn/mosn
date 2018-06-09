package handler

import (
	"context"
	"errors"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
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

//Add BOLTV2's Command Handler
func NewBoltCommandHandlerV2() *BoltCommandHandler {
	return &BoltCommandHandler{
		processors: map[int16]sofarpc.RemotingProcessor{
			sofarpc.RPC_REQUEST:  &BoltRequestProcessorV2{},
			sofarpc.RPC_RESPONSE: &BoltResponseProcessorV2{},
			sofarpc.HEARTBEAT:    &BoltHbProcessor{},
		},
	}
}

func (h *BoltCommandHandler) HandleCommand(context context.Context, msg interface{}, filter interface{}) error {
	logger := log.ByContext(context)
	
	if cmd, ok := msg.(sofarpc.ProtoBasicCmd); ok {
		cmdCode := cmd.GetCmdCode()
		
		if processor, ok := h.processors[cmdCode]; ok {
			//logger.Debugf("handle bolt command")
			processor.Process(context, cmd, filter)
		} else {
			errMsg := sofarpc.UnKnownCmdcode
			logger.Errorf(errMsg + "when decoding bolt %s", cmdCode)
			return errors.New(errMsg)
		}
	} else {
		errMsg := sofarpc.UnKnownCmd
		logger.Errorf(errMsg + "when decoding bolt %s", msg)
		return errors.New(errMsg)
	}
	
	return nil
}

func (h *BoltCommandHandler) RegisterProcessor(cmdCode int16, processor *sofarpc.RemotingProcessor) {
	if _, exists := h.processors[cmdCode]; exists {
		log.DefaultLogger.Warnf("bolt cmd handler [%x] alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}
