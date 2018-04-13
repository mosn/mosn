package handler

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize/hessian"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strconv"
	"gitlab.alipay-inc.com/afe/mosn/pkg/utility"
)

type TrResponseProcessor struct{}

func (b *TrResponseProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {

	if cmd, ok := msg.(*sofarpc.TrResponseCommand); ok {
		deserializeResponseAllFieldsTR(cmd)
		reqID := utility.StreamIDConvert(uint32(cmd.RequestID))

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				status := filter.OnDecodeHeader(reqID, cmd.ResponseHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.ResponseContent != nil {
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.ResponseContent))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

//Convert TR's Protocol Header  and Content Header to Map[string]string
func deserializeResponseAllFieldsTR(responseCommand *sofarpc.TrResponseCommand) {

	//DeSerialize Hessian
	hessianSerialize := hessian.HessianInstance
	ConnRequstBytes := responseCommand.ConnClassContent
	responseCommand.RequestID = hessianSerialize.SerializeConnResponseBytes(ConnRequstBytes)

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(responseCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader("requestFlag")] = strconv.FormatUint(uint64(responseCommand.RequestFlag), 10)
	allField[sofarpc.SofaPropertyHeader("serializeProtocol")] = strconv.FormatUint(uint64(responseCommand.SerializeProtocol), 10)
	allField[sofarpc.SofaPropertyHeader("direction")] = strconv.FormatUint(uint64(responseCommand.Direction), 10)
	allField[sofarpc.SofaPropertyHeader("reserved")] = strconv.FormatUint(uint64(responseCommand.Reserved), 10)
	allField[sofarpc.SofaPropertyHeader("appClassNameLen")] = strconv.FormatUint(uint64(responseCommand.AppClassNameLen), 10)
	allField[sofarpc.SofaPropertyHeader("connRequestLen")] = strconv.FormatUint(uint64(responseCommand.ConnRequestLen), 10)
	allField[sofarpc.SofaPropertyHeader("appClassContentLen")] = strconv.FormatUint(uint64(responseCommand.AppClassContentLen), 10)

	allField[sofarpc.SofaPropertyHeader("cmdcode")] = strconv.FormatUint(uint64(responseCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader("requestid")] = strconv.FormatUint(uint64(responseCommand.RequestID), 10)
	responseCommand.ResponseHeader = allField
}
