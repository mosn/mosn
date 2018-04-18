package handler

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/utility"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type BoltHbProcessor struct {
}

// ctx = type.serverStreamConnection
// CALLBACK STREAM LEVEL'S OnDecodeHeaders
func (b *BoltHbProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(cmd)
		reqID := utility.StreamIDConvert(cmd.ReqId)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				status := filter.OnDecodeHeader(reqID, cmd.RequestHeader)
				log.DefaultLogger.Debugf("Process Heartbeat Msg")
				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

