package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
)

type BoltResponseProcessor struct{}

func (b *BoltResponseProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltResponseCommand); ok {

		deserializeResponseAllFields(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.GetResponseHeader() != nil {
				//status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetRequestHeader())
				// 回调到stream中的OnDecoderHeader，回传HEADER数据
				status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetResponseHeader())

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

func deserializeResponseAllFields(responseCommand sofarpc.BoltResponseCommand) {
	//get instance
	serializeIns := serialize.Instance

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = string(responseCommand.GetProtocolCode())
	allField[sofarpc.SofaPropertyHeader("cmdType")] = string(responseCommand.GetCmdType())
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = sofarpc.UintToString(uint16(responseCommand.GetCmdCode()), 16)
	allField[sofarpc.SofaPropertyHeader("version")] = string(responseCommand.GetVersion())
	allField[sofarpc.SofaPropertyHeader("requestId")] = sofarpc.UintToString(uint32(responseCommand.GetId()), 32)
	allField[sofarpc.SofaPropertyHeader("codec")] = string(responseCommand.GetCodec())
	allField[sofarpc.SofaPropertyHeader("classLength")] = sofarpc.UintToString(uint16(responseCommand.GetClassLength()), 16)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = sofarpc.UintToString(uint16(responseCommand.GetHeaderLength()), 16)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = sofarpc.UintToString(uint32(responseCommand.GetCmdCode()), 32)

	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	allField[sofarpc.SofaPropertyHeader("responseStatus")] = sofarpc.UintToString(uint16(responseCommand.GetResponseStatus()), 16)
	//暂时不知道responseTimeMills封装在协议的位置
	allField[sofarpc.SofaPropertyHeader("responseTimeMills")] = sofarpc.UintToString(uint64(responseCommand.GetResponseTimeMillis()), 64)

	//serialize class name
	var className string
	serializeIns.DeSerialize(responseCommand.GetClass(), &className)
	allField[sofarpc.SofaPropertyHeader("className")] = className

	//serialize header
	var headerMap map[string]string
	serializeIns.DeSerialize(responseCommand.GetHeader(), &headerMap)
	fmt.Println("deSerialize  headerMap:", headerMap)

	for k, v := range headerMap {
		allField[k] = v
	}

	responseCommand.SetResponseHeader(allField)
}
