package handler

import (
	"context"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type BoltHbProcessor struct {
}

// ctx = type.serverStreamConnection
func (b *BoltHbProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	logger := log.ByContext(context)

	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(context, cmd)
		reqID := sofarpc.StreamIDConvert(cmd.ReqId)

		//Heartbeat message only has request header
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				status := filter.OnDecodeHeader(reqID, cmd.RequestHeader)
		//		logger.Debugf("Process Heartbeat Request Msg")

				if status == types.StopIteration {
					return
				}
			}
		}
	} else if cmd, ok := msg.(*sofarpc.BoltResponseCommand); ok {
		deserializeResponseAllFields(cmd, context)
		reqID := sofarpc.StreamIDConvert(cmd.ReqId)
		//logger := log.ByContext(context)

		//Heartbeat message only has request header
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				status := filter.OnDecodeHeader(reqID, cmd.ResponseHeader)
			//	logger.Debugf("Process Heartbeat Response Msg")

				if status == types.StopIteration {
					return
				}
			}
		}
	} else {
		logger.Errorf("decode heart beat error\n")
	}
}
