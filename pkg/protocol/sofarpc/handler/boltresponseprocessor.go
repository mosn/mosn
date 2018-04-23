package handler

import (
	"context"
	"strconv"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type BoltResponseProcessor struct{}
type BoltResponseProcessorV2 struct{}

func (b *BoltResponseProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltResponseCommand); ok {
		deserializeResponseAllFields(cmd, context)
		reqID := sofarpc.StreamIDConvert(cmd.ReqId)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {
				// 回调到stream中的OnDecoderHeader，回传HEADER数据
				status := filter.OnDecodeHeader(reqID, cmd.ResponseHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

func (b *BoltResponseProcessorV2) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltV2ResponseCommand); ok {
		deserializeResponseAllFieldsV2(cmd, context)
		reqID := sofarpc.StreamIDConvert(cmd.ReqId)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.ResponseHeader != nil {

				status := filter.OnDecodeHeader(reqID, cmd.ResponseHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				///回调到stream中的OnDecoderDATA，回传CONTENT数据
				status := filter.OnDecodeData(reqID, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}
func deserializeResponseAllFields(responseCommand *sofarpc.BoltResponseCommand, context context.Context) {
	//get instance
	serializeIns := serialize.Instance

	//serialize header
	headerMap := sofarpc.GetMap(context, 64)
	serializeIns.DeSerialize(responseCommand.HeaderMap, &headerMap)
	log.DefaultLogger.Debugf("deSerialize  headerMap:", headerMap)

	allField := sofarpc.GetMap(context, 20+len(headerMap))

	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(responseCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(responseCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(responseCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(responseCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(responseCommand.ReqId), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(responseCommand.CodecPro), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(responseCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(responseCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(responseCommand.ContentLen), 10)
	// FOR RESPONSE,ENCODE RESPONSE STATUS and RESPONSE TIME
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespStatus)] = strconv.FormatUint(uint64(responseCommand.ResponseStatus), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderRespTimeMills)] = strconv.FormatUint(uint64(responseCommand.ResponseTimeMillis), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(responseCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = className
	log.DefaultLogger.Debugf("Response ClassName is:", className)

	for k, v := range headerMap {
		allField[k] = v
	}

	sofarpc.ReleaseMap(context, headerMap)

	responseCommand.ResponseHeader = allField
}

func deserializeResponseAllFieldsV2(responseCommandV2 *sofarpc.BoltV2ResponseCommand, context context.Context) {
	deserializeResponseAllFields(&responseCommandV2.BoltResponseCommand, context)
	responseCommandV2.ResponseHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(responseCommandV2.Version1), 10)
	responseCommandV2.ResponseHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(responseCommandV2.SwitchCode), 10)
}
