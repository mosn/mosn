package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
)

type BoltRequestProcessor struct{}

// ctx = type.serverStreamConnection
func (b *BoltRequestProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.GetRequestHeader() != nil {
				//status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetRequestHeader())
				// 回调到stream中的OnDecoderHeader，回传HEADER数据
				status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetRequestHeader())

				if status == types.StopIteration {
					return
				}
			}

			if cmd.GetContent() != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(cmd.GetId(), buffer.NewIoBufferBytes(cmd.GetContent()))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

//  将所有BOLT的HEADER字段组装成map结构
func deserializeRequestAllFields(requestCommand sofarpc.BoltRequestCommand) {
	//get instance
	serializeIns := serialize.Instance

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = string(requestCommand.GetProtocolCode())
	allField[sofarpc.SofaPropertyHeader("cmdType")] = string(requestCommand.GetCmdType())
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = sofarpc.UintToString(uint16(requestCommand.GetCmdCode()), 16)
	allField[sofarpc.SofaPropertyHeader("version")] = string(requestCommand.GetVersion())
	allField[sofarpc.SofaPropertyHeader("requestId")] = sofarpc.UintToString(uint32(requestCommand.GetId()), 32)
	allField[sofarpc.SofaPropertyHeader("codec")] = string(requestCommand.GetCodec())
	allField[sofarpc.SofaPropertyHeader("timeout")] = sofarpc.UintToString(uint32(requestCommand.GetTimeout()), 32)
	allField[sofarpc.SofaPropertyHeader("classLength")] = sofarpc.UintToString(uint16(requestCommand.GetClassLength()), 16)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = sofarpc.UintToString(uint16(requestCommand.GetHeaderLength()), 16)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = sofarpc.UintToString(uint32(requestCommand.GetContentLength()), 32)

	//serialize class name
	var className string
	serializeIns.DeSerialize(requestCommand.GetClass(), &className)
	allField[sofarpc.SofaPropertyHeader("className")] = className

	//serialize header
	var headerMap map[string]string
	serializeIns.DeSerialize(requestCommand.GetHeader(), &headerMap)
	fmt.Println("deSerialize  headerMap:", headerMap)

	for k, v := range headerMap {
		allField[k] = v
	}

	requestCommand.SetRequestHeader(allField)
}
