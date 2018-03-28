package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strconv"
)

type BoltRequestProcessor struct{}

type BoltRequestProcessorV2 struct{}

// ctx = type.serverStreamConnection
func (b *BoltRequestProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFields(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				status := filter.OnDecodeHeader(cmd.ReqId, cmd.RequestHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(cmd.ReqId, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

// ctx = type.serverStreamConnection
func (b *BoltRequestProcessorV2) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltRequestCommandV2); ok {
		deserializeRequestAllFieldsV2(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.RequestHeader != nil {
				status := filter.OnDecodeHeader(cmd.ReqId, cmd.RequestHeader)

				if status == types.StopIteration {
					return
				}
			}

			if cmd.Content != nil {
				status := filter.OnDecodeData(cmd.ReqId, buffer.NewIoBufferBytes(cmd.Content))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

//Convert BoltV1's Protocol Header  and Content Header to Map[string]string
func deserializeRequestAllFields(requestCommand sofarpc.BoltRequestCommand) {
	//get instance
	serializeIns := serialize.Instance

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(requestCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader("cmdType")] = strconv.FormatUint(uint64(requestCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = strconv.FormatUint(uint64(requestCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader("version")] = strconv.FormatUint(uint64(requestCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader("requestId")] = strconv.FormatUint(uint64(requestCommand.ReqId), 10)
	allField[sofarpc.SofaPropertyHeader("codec")] = strconv.FormatUint(uint64(requestCommand.CodecPro), 10)
	allField[sofarpc.SofaPropertyHeader("timeout")] = strconv.FormatUint(uint64(requestCommand.Timeout), 10)
	allField[sofarpc.SofaPropertyHeader("classLength")] = strconv.FormatUint(uint64(requestCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = strconv.FormatUint(uint64(requestCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = strconv.FormatUint(uint64(requestCommand.ContentLen), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(requestCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader("className")] = className

	//serialize header
	var headerMap map[string]string
	serializeIns.DeSerialize(requestCommand.HeaderMap, &headerMap)
	fmt.Println("deSerialize  headerMap:", headerMap)

	for k, v := range headerMap {
		allField[k] = v
	}

	requestCommand.RequestHeader = allField
}

//  将所有BOLT的HEADER字段组装成map结构
func deserializeRequestAllFieldsV2(requestCommand sofarpc.BoltRequestCommandV2) {
	//get instance
	serializeIns := serialize.Instance

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(requestCommand.Protocol), 10)
	allField[sofarpc.SofaPropertyHeader("cmdType")] = strconv.FormatUint(uint64(requestCommand.CmdType), 10)
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = strconv.FormatUint(uint64(requestCommand.CmdCode), 10)
	allField[sofarpc.SofaPropertyHeader("version")] = strconv.FormatUint(uint64(requestCommand.Version), 10)
	allField[sofarpc.SofaPropertyHeader("requestId")] = strconv.FormatUint(uint64(requestCommand.ReqId), 10)
	allField[sofarpc.SofaPropertyHeader("codec")] = strconv.FormatUint(uint64(requestCommand.CodecPro), 10)
	allField[sofarpc.SofaPropertyHeader("timeout")] = strconv.FormatUint(uint64(requestCommand.Timeout), 10)
	allField[sofarpc.SofaPropertyHeader("classLength")] = strconv.FormatUint(uint64(requestCommand.ClassLen), 10)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = strconv.FormatUint(uint64(requestCommand.HeaderLen), 10)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = strconv.FormatUint(uint64(requestCommand.ContentLen), 10)

	allField[sofarpc.SofaPropertyHeader("ver1")] = strconv.FormatUint(uint64(requestCommand.Version1), 10)
	allField[sofarpc.SofaPropertyHeader("switchcode")] = strconv.FormatUint(uint64(requestCommand.SwitchCode), 10)

	//serialize class name
	var className string
	serializeIns.DeSerialize(requestCommand.ClassName, &className)
	allField[sofarpc.SofaPropertyHeader("className")] = className

	//serialize header
	var headerMap map[string]string
	serializeIns.DeSerialize(requestCommand.HeaderMap, &headerMap)
	fmt.Println("deSerialize  headerMap:", headerMap)

	for k, v := range headerMap {
		allField[k] = v
	}

	requestCommand.RequestHeader = allField
}
