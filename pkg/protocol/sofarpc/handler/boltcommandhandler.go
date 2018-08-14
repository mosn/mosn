/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package handler

import (
	"context"
	"errors"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/sofarpc"
)

type boltCommandHandler struct {
	processors map[int16]sofarpc.RemotingProcessor
}

// NewBoltCommandHandler
// new bolt command handler
func NewBoltCommandHandler() sofarpc.CommandHandler {
	return &boltCommandHandler{
		processors: map[int16]sofarpc.RemotingProcessor{
			sofarpc.RPC_REQUEST:  &BoltRequestProcessor{},
			sofarpc.RPC_RESPONSE: &BoltResponseProcessor{},
			sofarpc.HEARTBEAT:    &boltHbProcessor{},
		},
	}
}

// NewBoltCommandHandlerV2
// Add BOLTV2's Command Handler
func NewBoltCommandHandlerV2() sofarpc.CommandHandler {
	return &boltCommandHandler{
		processors: map[int16]sofarpc.RemotingProcessor{
			sofarpc.RPC_REQUEST:  &BoltRequestProcessorV2{},
			sofarpc.RPC_RESPONSE: &BoltResponseProcessorV2{},
			sofarpc.HEARTBEAT:    &boltHbProcessor{},
		},
	}
}

func (h *boltCommandHandler) HandleCommand(context context.Context, msg interface{}, filter interface{}) error {
	logger := log.ByContext(context)

	if cmd, ok := msg.(sofarpc.ProtoBasicCmd); ok {
		cmdCode := cmd.GetCmdCode()

		if processor, ok := h.processors[cmdCode]; ok {
			//logger.Debugf("handle bolt command")
			processor.Process(context, cmd, filter)
		} else {
			errMsg := sofarpc.UnKnownCmdcode
			logger.Errorf(errMsg+"when decoding bolt %s", cmdCode)
			return errors.New(errMsg)
		}
	} else {
		errMsg := sofarpc.UnKnownCmd
		logger.Errorf(errMsg+"when decoding bolt %s", msg)
		return errors.New(errMsg)
	}

	return nil
}

func (h *boltCommandHandler) RegisterProcessor(cmdCode int16, processor *sofarpc.RemotingProcessor) {
	if _, exists := h.processors[cmdCode]; exists {
		log.DefaultLogger.Warnf("bolt cmd handler [%x] already exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}
