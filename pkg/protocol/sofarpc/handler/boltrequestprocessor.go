package handler

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type BoltRequestProcessor struct {
}

func (b *BoltRequestProcessor) Process(ctx interface{}, msg interface{}, executor interface{}) {
	if cmd, ok := msg.(sofarpc.BoltRequestCommand); ok {
		deserializeRequest(cmd)

		//for demo, invoke ctx as callback
		if filter, ok := ctx.(types.DecodeFilter); ok {
			if cmd.GetRequestHeader() != nil {
				status := filter.OnDecodeHeader(cmd.GetRequestHeader())

				if status == types.StopIteration {
					return
				}
			}
		}
	}
}

// 反序列化请求
func deserializeRequest(requestCommand sofarpc.BoltRequestCommand) (sofarpc.BoltRequestCommand, error) {
	//get instance
	serialize := serialize.Instance

	var clazzName string
	serialize.DeSerialize(requestCommand.GetClass(), &clazzName)

	fmt.Println("deSerialize clazz :", clazzName)

	var headerMap map[string]string
	serialize.DeSerialize(requestCommand.GetHeader(), &headerMap)

	fmt.Println("deSerialize  headerMap:", headerMap)
	requestCommand.SetRequestHeader(headerMap)

	return requestCommand, nil
}
