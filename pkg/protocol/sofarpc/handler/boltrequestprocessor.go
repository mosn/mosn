package handler

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/serialize"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc/codec"
)

type BoltRequestProcessor struct {
}

func (b *BoltRequestProcessor) Process(ctx interface{}, msg interface{}, executor interface{}){
	if cmd, ok := msg.(*codec.BoltRequestCommand); ok {
		deserializeRequest(cmd)
	}

}


 // 反序列化请求
func deserializeRequest(requestCommand *codec.BoltRequestCommand) (*codec.BoltRequestCommand, error) {
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