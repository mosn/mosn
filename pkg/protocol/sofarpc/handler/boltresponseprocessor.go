package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strconv"
)

type BoltResponseProcessor struct{}
type BoltResponseProcessorV2 struct{}

func (b *BoltResponseProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltResponseCommand); ok {
		deserializeResponseAllFields(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				//status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetRequestHeader())
				// 回调到stream中的OnDecoderHeader，回传HEADER数据
				status := filter.OnDecodeHeader(cmd.ReqId, cmd.ResponseHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(cmd.ReqId, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}
func (b *BoltResponseProcessorV2) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltResponseCommandV2); ok {
		deserializeResponseAllFieldsV2(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {

				status := filter.OnDecodeHeader(cmd.ReqId, cmd.ResponseHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(cmd.ReqId, buffer.NewIoBufferBytes(cmd.Content))

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
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(responseCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader("cmdType")] = strconv.FormatUint(uint64(responseCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = strconv.FormatUint(uint64(responseCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader("version")] = strconv.FormatUint(uint64(responseCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader("requestId")] = strconv.FormatUint(uint64(responseCommand.ReqId), 10)
	allField[sofarpc.SofaPropertyHeader("codec")] = strconv.FormatUint(uint64(responseCommand.CodecPro), 10)

	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	allField[sofarpc.SofaPropertyHeader("responseStatus")] = strconv.FormatUint(uint64(responseCommand.ResponseStatus), 10)

	allField[sofarpc.SofaPropertyHeader("classLength")] = strconv.FormatUint(uint64(responseCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = strconv.FormatUint(uint64(responseCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = strconv.FormatUint(uint64(responseCommand.ContentLen), 10)
	//暂时不知道responseTimeMills封装在协议的位置
	allField[sofarpc.SofaPropertyHeader("responseTimeMills")] = strconv.FormatUint(uint64(responseCommand.ResponseTimeMillis), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(responseCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader("className")] = className

	//serialize header
	var headerMap map[string]string
	serializeIns.DeSerialize(responseCommand.HeaderMap, &headerMap)
	fmt.Println("deSerialize  headerMap:", headerMap)

	for k, v := range headerMap {
		allField[k] = v
	}

	responseCommand.ResponseHeader = allField
}

func deserializeResponseAllFieldsV2(responseCommand sofarpc.BoltResponseCommandV2) {
	//get instance
	serializeIns := serialize.Instance

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(responseCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader("cmdType")] = strconv.FormatUint(uint64(responseCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = strconv.FormatUint(uint64(responseCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader("version")] = strconv.FormatUint(uint64(responseCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader("requestId")] = strconv.FormatUint(uint64(responseCommand.ReqId), 10)
	allField[sofarpc.SofaPropertyHeader("codec")] = strconv.FormatUint(uint64(responseCommand.CodecPro), 10)

	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	allField[sofarpc.SofaPropertyHeader("responseStatus")] = strconv.FormatUint(uint64(responseCommand.ResponseStatus), 10)

	allField[sofarpc.SofaPropertyHeader("classLength")] = strconv.FormatUint(uint64(responseCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = strconv.FormatUint(uint64(responseCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = strconv.FormatUint(uint64(responseCommand.ContentLen), 10)
	//暂时不知道responseTimeMills封装在协议的位置
	allField[sofarpc.SofaPropertyHeader("responseTimeMills")] = strconv.FormatUint(uint64(responseCommand.ResponseTimeMillis), 10)

	allField[sofarpc.SofaPropertyHeader("ver1")] = strconv.FormatUint(uint64(responseCommand.Version1), 10)
	allField[sofarpc.SofaPropertyHeader("switchcode")] = strconv.FormatUint(uint64(responseCommand.SwitchCode), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(responseCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader("className")] = className

	//serialize header
	var headerMap map[string]string
	serializeIns.DeSerialize(responseCommand.HeaderMap, &headerMap)
	fmt.Println("deSerialize  headerMap:", headerMap)

	for k, v := range headerMap {
		allField[k] = v
	}

	responseCommand.ResponseHeader = allField
}
