package handler

import (
	"context"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type BoltHbProcessor struct {
}

// ctx = type.serverStreamConnection
// CALLBACK STREAM LEVEL'S OnDecodeHeaders
func (b *BoltHbProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(context, cmd)
		reqID := sofarpc.StreamIDConvert(cmd.ReqId)
		logger := log.ByContext(context)

		//HeartBeat Msg only has request header
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				//CALLBACK STREAM LEVEL'S ON_DECODE_HEADER
				status := filter.OnDecodeHeader(reqID, cmd.RequestHeader)
				logger.Debugf("Process Heartbeat Msg")

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}
