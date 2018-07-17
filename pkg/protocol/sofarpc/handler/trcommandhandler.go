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

func (h *TrCommandHandler) HandleCommand(context context.Context, msg interface{}, filter interface{}) error {
	if cmd, ok := msg.(sofarpc.ProtoBasicCmd); ok {
		cmdCode := cmd.GetCmdCode()
		logger := log.ByContext(context)

		if processor, ok := h.processors[cmdCode]; ok {
			logger.Debugf("handle tr command")

			processor.Process(context, cmd, filter)
		} else {
			errMsg := sofarpc.UnKnownCmdcode
			logger.Errorf(errMsg+"when decoding tr %s", cmdCode)
			return errors.New(errMsg)
		}
	}
	return nil
}

func (h *TrCommandHandler) RegisterProcessor(cmdCode int16, processor *sofarpc.RemotingProcessor) {
	if _, exists := h.processors[cmdCode]; exists {
		log.DefaultLogger.Warnf("tr cmd handler [%x] alreay exist:", cmdCode)
	} else {
		h.processors[cmdCode] = *processor
	}
}
