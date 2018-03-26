package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"strconv"
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
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(responseCommand.GetProtocolCode()), 10)
	allField[sofarpc.SofaPropertyHeader("cmdType")] = strconv.FormatUint(uint64(responseCommand.GetCmdType()), 10)
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = strconv.FormatUint(uint64(responseCommand.GetCmdCode()), 10)
	allField[sofarpc.SofaPropertyHeader("version")] = strconv.FormatUint(uint64(responseCommand.GetVersion()), 10)
	allField[sofarpc.SofaPropertyHeader("requestId")] = strconv.FormatUint(uint64(responseCommand.GetId()), 10)
	allField[sofarpc.SofaPropertyHeader("codec")] = strconv.FormatUint(uint64(responseCommand.GetCodec()), 10)
	allField[sofarpc.SofaPropertyHeader("classLength")] = strconv.FormatUint(uint64(responseCommand.GetClassLength()), 10)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = strconv.FormatUint(uint64(responseCommand.GetHeaderLength()), 10)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = strconv.FormatUint(uint64(responseCommand.GetContentLength()), 10)
	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	allField[sofarpc.SofaPropertyHeader("responseStatus")] = strconv.FormatUint(uint64(responseCommand.GetResponseStatus()), 10)
	//暂时不知道responseTimeMills封装在协议的位置
	allField[sofarpc.SofaPropertyHeader("responseTimeMills")] = strconv.FormatUint(uint64(responseCommand.GetResponseTimeMillis()), 10)

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
