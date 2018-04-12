package handler

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize/hessian"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/utility"
	"strconv"
)

type TrRequestProcessor struct{}

// ctx = type.serverStreamConnection
// CALLBACK STREAM LEVEL'S OnDecodeHeaders
func (b *TrRequestProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(*sofarpc.TrRequestCommand); ok {
		deserializeRequestAllFieldsTR(cmd)
		reqID := utility.StreamIDConvert(uint32(cmd.RequestID))

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				status := filter.OnDecodeHeader(reqID, cmd.RequestHeader)
				if status == types.StopIteration {
					return
				}
			}

			if cmd.RequestContent != nil {
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.RequestContent))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

//Convert TR's Protocol Header  and Content Header to Map[string]string
func deserializeRequestAllFieldsTR(requestCommand *sofarpc.TrRequestCommand) {

	//DeSerialize Hessian
	hessianSerialize := hessian.HessianInstance

	ConnRequstBytes := requestCommand.ConnClassContent
	AppRequstBytes := requestCommand.AppClassContent

	requestCommand.RequestID = hessianSerialize.SerializeConnRequestBytes(ConnRequstBytes)
	requestCommand.TargetServiceUniqueName = hessianSerialize.SerializeAppRequestBytes(AppRequstBytes)

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(requestCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader("requestflag")] = strconv.FormatUint(uint64(requestCommand.RequestFlag), 10)
	allField[sofarpc.SofaPropertyHeader("serializeprotocol")] = strconv.FormatUint(uint64(requestCommand.SerializeProtocol), 10)
	allField[sofarpc.SofaPropertyHeader("direction")] = strconv.FormatUint(uint64(requestCommand.Direction), 10)
	allField[sofarpc.SofaPropertyHeader("reserved")] = strconv.FormatUint(uint64(requestCommand.Reserved), 10)
	allField[sofarpc.SofaPropertyHeader("appclassnamelen")] = strconv.FormatUint(uint64(requestCommand.AppClassNameLen), 10)
	allField[sofarpc.SofaPropertyHeader("connrequestlen")] = strconv.FormatUint(uint64(requestCommand.ConnRequestLen), 10)
	allField[sofarpc.SofaPropertyHeader("appclasscontentlen")] = strconv.FormatUint(uint64(requestCommand.AppClassContentLen), 10)

	allField[sofarpc.SofaPropertyHeader("cmdcode")] = strconv.FormatUint(uint64(requestCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader("requestid")] = strconv.FormatUint(uint64(requestCommand.RequestID), 10)

	//TargetServiceUniqueName
	allField["service"] = requestCommand.TargetServiceUniqueName

	requestCommand.RequestHeader = allField
}
