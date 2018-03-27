package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
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
			if cmd.GetRequestHeader() != nil {
				status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetRequestHeader())

				if status == types.StopIteration {
					return
				}
			}

			if cmd.GetContent() != nil {
				status := filter.OnDecodeData(cmd.GetId(), buffer.NewIoBufferBytes(cmd.GetContent()))

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

// ctx = type.serverStreamConnection
func (b *BoltRequestProcessorV2) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltRequestCommand); ok {
		deserializeRequestAllFieldsV2(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.GetRequestHeader() != nil {
				status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetRequestHeader())

				if status == types.StopIteration {
					return
				}
			}

			if cmd.GetContent() != nil {
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
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(requestCommand.GetProtocolCode()), 10)
	allField[sofarpc.SofaPropertyHeader("cmdType")] = strconv.FormatUint(uint64(requestCommand.GetCmdType()), 10)
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = strconv.FormatUint(uint64(requestCommand.GetCmdCode()), 10)
	allField[sofarpc.SofaPropertyHeader("version")] = strconv.FormatUint(uint64(requestCommand.GetVersion()), 10)
	allField[sofarpc.SofaPropertyHeader("requestId")] = strconv.FormatUint(uint64(requestCommand.GetId()), 10)
	allField[sofarpc.SofaPropertyHeader("codec")] = strconv.FormatUint(uint64(requestCommand.GetCodec()), 10)
	allField[sofarpc.SofaPropertyHeader("timeout")] = strconv.FormatUint(uint64(requestCommand.GetTimeout()), 10)
	allField[sofarpc.SofaPropertyHeader("classLength")] = strconv.FormatUint(uint64(requestCommand.GetClassLength()), 10)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = strconv.FormatUint(uint64(requestCommand.GetHeaderLength()), 10)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = strconv.FormatUint(uint64(requestCommand.GetContentLength()), 10)

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


//  将所有BOLT的HEADER字段组装成map结构
func deserializeRequestAllFieldsV2(requestCommand sofarpc.BoltRequestCommand) {
	//get instance
	serializeIns := serialize.Instance

	allField := map[string]string{}
	allField[sofarpc.SofaPropertyHeader("protocol")] = strconv.FormatUint(uint64(requestCommand.GetProtocolCode()), 10)
	allField[sofarpc.SofaPropertyHeader("cmdType")] = strconv.FormatUint(uint64(requestCommand.GetCmdType()), 10)
	allField[sofarpc.SofaPropertyHeader("cmdCode")] = strconv.FormatUint(uint64(requestCommand.GetCmdCode()), 10)
	allField[sofarpc.SofaPropertyHeader("version")] = strconv.FormatUint(uint64(requestCommand.GetVersion()), 10)
	allField[sofarpc.SofaPropertyHeader("requestId")] = strconv.FormatUint(uint64(requestCommand.GetId()), 10)
	allField[sofarpc.SofaPropertyHeader("codec")] = strconv.FormatUint(uint64(requestCommand.GetCodec()), 10)
	allField[sofarpc.SofaPropertyHeader("timeout")] = strconv.FormatUint(uint64(requestCommand.GetTimeout()), 10)
	allField[sofarpc.SofaPropertyHeader("classLength")] = strconv.FormatUint(uint64(requestCommand.GetClassLength()), 10)
	allField[sofarpc.SofaPropertyHeader("headerLength")] = strconv.FormatUint(uint64(requestCommand.GetHeaderLength()), 10)
	allField[sofarpc.SofaPropertyHeader("contentLength")] = strconv.FormatUint(uint64(requestCommand.GetContentLength()), 10)

	allField[sofarpc.SofaPropertyHeader("ver1")] = strconv.FormatUint(uint64(requestCommand.GetVer1()), 10)
	allField[sofarpc.SofaPropertyHeader("switchcode")] = strconv.FormatUint(uint64(requestCommand.GetSwitch()), 10)

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