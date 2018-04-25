package handler

import (
	"context"
	"strconv"
	"sync/atomic"

	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

var streamIdCsounter uint32
var defaultTmpBufferSize = 1 << 6

type BoltRequestProcessor struct{}

type BoltRequestProcessorV2 struct{}

// ctx = type.serverStreamConnection
// CALLBACK STREAM LEVEL'S OnDecodeHeaders
func (b *BoltRequestProcessor) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(context, cmd)
		streamId := atomic.AddUint32(&streamIdCsounter, 1)
		streamIdStr := sofarpc.StreamIDConvert(streamId)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				//CALLBACK STREAM LEVEL'S ONDECODEHEADER
				status := filter.OnDecodeHeader(streamIdStr, cmd.RequestHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamIdStr, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

// ctx = type.serverStreamConnection
func (b *BoltRequestProcessorV2) Process(context context.Context, msg interface{}, filter interface{}) {
	if cmd, ok := msg.(*sofarpc.BoltV2RequestCommand); ok {
		deserializeRequestAllFieldsV2(cmd, context)
		streamId := atomic.AddUint32(&streamIdCsounter, 1)
		streamIdStr := sofarpc.StreamIDConvert(streamId)

		//for demo, invoke ctx as callback
		if filter, ok := filter.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				status := filter.OnDecodeHeader(streamIdStr, cmd.RequestHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(streamIdStr, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

//Convert BoltV1's Protocol Header  and Content Header to Map[string]string
func deserializeRequestAllFields(context context.Context, requestCommand *sofarpc.BoltRequestCommand) {
	//get instance
	serializeIns := serialize.Instance

	//serialize header
	headerMap := sofarpc.GetMap(context, defaultTmpBufferSize)

	//logger
	logger := log.ByContext(context)


	serializeIns.DeSerialize(requestCommand.HeaderMap, &headerMap)
	logger.Debugf("deSerialize  headerMap:", headerMap)

	allField := sofarpc.GetMap(context, 20+len(headerMap))
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderProtocolCode)] = strconv.FormatUint(uint64(requestCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdType)] = strconv.FormatUint(uint64(requestCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCmdCode)] = strconv.FormatUint(uint64(requestCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion)] = strconv.FormatUint(uint64(requestCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderReqID)] = strconv.FormatUint(uint64(requestCommand.ReqId), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderCodec)] = strconv.FormatUint(uint64(requestCommand.CodecPro), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderTimeout)] = strconv.FormatUint(uint64(requestCommand.Timeout), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassLen)] = strconv.FormatUint(uint64(requestCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderHeaderLen)] = strconv.FormatUint(uint64(requestCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderContentLen)] = strconv.FormatUint(uint64(requestCommand.ContentLen), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(requestCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader(sofarpc.HeaderClassName)] = className
	logger.Debugf("Request ClassName is:", className)

	for k, v := range headerMap {
		allField[k] = v
	}

	sofarpc.ReleaseMap(context, headerMap)

	requestCommand.RequestHeader = allField
}

func deserializeRequestAllFieldsV2(requestCommandV2 *sofarpc.BoltV2RequestCommand, context context.Context) {
	deserializeRequestAllFields(context, &requestCommandV2.BoltRequestCommand)
	requestCommandV2.RequestHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderVersion1)] = strconv.FormatUint(uint64(requestCommandV2.Version1), 10)
	requestCommandV2.RequestHeader[sofarpc.SofaPropertyHeader(sofarpc.HeaderSwitchCode)] = strconv.FormatUint(uint64(requestCommandV2.SwitchCode), 10)
}
