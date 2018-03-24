package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"




)

type BoltRequestProcessor struct {

}


// ctx = type.serverStreamConnection
func (b *BoltRequestProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltRequestCommand); ok {
		//deserializeRequestHeaders(cmd)    //做反序列化

		allHeader := deserializeRequestAllFields(cmd)
		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.GetRequestHeader() != nil {
				//status := filter.OnDecodeHeader(cmd.GetId(), cmd.GetRequestHeader())
				// 回调到stream中的OnDecoderHeader，回传HEADER数据
				status := filter.OnDecodeHeader(cmd.GetId(),allHeader)

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

// 反序列化请求
func deserializeRequestHeaders(requestCommand sofarpc.BoltRequestCommand) (sofarpc.BoltRequestCommand, error){
	//get instance
	serialize := serialize.Instance

	var className string
	serialize.DeSerialize(requestCommand.GetClass(), &className)

	fmt.Println("deSerialize class :", className)

	var headerMap map[string]string
	serialize.DeSerialize(requestCommand.GetHeader(), &headerMap)

	fmt.Println("deSerialize  headerMap:", headerMap)
	requestCommand.SetRequestHeader(headerMap)    //SET Map[]

	return requestCommand, nil
}


//  将所有BOLT的HEADER字段组装成map结构
func deserializeRequestAllFields(requestCommand sofarpc.BoltRequestCommand) map[string]string{
	//get instance
	serializeIns := serialize.Instance

	allField := map[string]string{}

	allField["XXX_protocol"] = string(requestCommand.GetProtocolCode())
	allField["XXX_cmdType"]  = string(requestCommand.GetCmdType())
	allField["XXX_cmdCode"] = codec.UintToString(uint16(requestCommand.GetCmdCode()),16)
	allField["XXX_version"] = string(requestCommand.GetVersion())
	allField["XXX_requestId"] = codec.UintToString(uint32(requestCommand.GetId()),32)
	allField["XXX_codec"]  = string(requestCommand.GetCodec())

	allField["XXX_timeout"] = codec.UintToString(requestCommand.GetTimeout(),32)

	allField["XXX_classLength"] = codec.UintToString(uint16(requestCommand.GetClassLength()),16)
	allField["XXX_headerLength"] = codec.UintToString(uint16(requestCommand.GetHeaderLength()),16)
	allField["XXX_contentLength"] = codec.UintToString(uint32(requestCommand.GetCmdCode()),32)


	//serialize class name
	var className string
	serializeIns.DeSerialize(requestCommand.GetClass(), &className)
	fmt.Println("deSerialize class :", className)
	allField["XXX_className"]  = className


	//serialize header
	var headerMap map[string]string
	serializeIns.DeSerialize(requestCommand.GetHeader(), &headerMap)
	fmt.Println("deSerialize  headerMap:", headerMap)
	requestCommand.SetRequestHeader(headerMap)

	for k,v := range headerMap {

		allField[k] = v
	}

	return allField
}


