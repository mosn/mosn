package handler

import (
	"context"
	"strconv"

	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize/hessian"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type TrResponseProcessor struct{}

func (b *TrResponseProcessor) Process(context context.Context, msg interface{}, filter interface{}) {

	if cmd, ok := msg.(*sofarpc.TrResponseCommand); ok {
		deserializeResponseAllFieldsTR(cmd, context)
		reqID := sofarpc.StreamIDConvert(uint32(cmd.RequestID))

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
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
func deserializeResponseAllFieldsTR(responseCommand *sofarpc.TrResponseCommand, context context.Context) {

	//DeSerialize Hessian
	hessianSerialize := hessian.HessianInstance
	ConnRequstBytes := responseCommand.ConnClassContent
	responseCommand.RequestID = hessianSerialize.SerializeConnResponseBytes(ConnRequstBytes)

	allField := sofarpc.GetMap(context, 20)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(responseCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqFlag)] = strconv.FormatUint(uint64(responseCommand.RequestFlag), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderSeriProtocol)] = strconv.FormatUint(uint64(responseCommand.SerializeProtocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderDirection)] = strconv.FormatUint(uint64(responseCommand.Direction), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReserved)] = strconv.FormatUint(uint64(responseCommand.Reserved), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderAppclassnamelen)] = strconv.FormatUint(uint64(responseCommand.AppClassNameLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderConnrequestlen)] = strconv.FormatUint(uint64(responseCommand.ConnRequestLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderAppclasscontentlen)] = strconv.FormatUint(uint64(responseCommand.AppClassContentLen), 10)

	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(responseCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(responseCommand.RequestID), 10)

	responseCommand.ResponseHeader = allField
}
