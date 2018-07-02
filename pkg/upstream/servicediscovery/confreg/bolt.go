package registry

import (
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"strconv"
)

const PublisherClassName = "com.alipay.sofa.registry.common.model.client.pb.PublisherRegisterPb"
const SubscriberClassName = "com.alipay.sofa.registry.common.model.client.pb.SubscriberRegisterPb"

var PublisherClassLen = strconv.Itoa(len(PublisherClassName))
var SubscriberClassLen = strconv.Itoa(len(SubscriberClassName))

func BuildBoltPublishRequestCommand(contentLen int, reqId string) map[string]string {
	headers := buildBasicRequestCommand()
	headers[sofarpc.HeaderReqID] = reqId
	headers[sofarpc.HeaderClassName] = PublisherClassName
	headers[sofarpc.HeaderClassLen] = PublisherClassLen
	headers[sofarpc.HeaderContentLen] = fmt.Sprintf("%d", contentLen)

	return headers
}

func BuildBoltSubscribeRequestCommand(contentLen int, reqId string) map[string]string {
	headers := buildBasicRequestCommand()
	headers[sofarpc.HeaderReqID] = reqId
	headers[sofarpc.HeaderClassName] = SubscriberClassName
	headers[sofarpc.HeaderClassLen] = SubscriberClassLen
	headers[sofarpc.HeaderContentLen] = fmt.Sprintf("%d", contentLen)

	return headers
}

func buildBasicRequestCommand() map[string]string {
	headers := make(map[string]string)
	headers[sofarpc.HeaderProtocolCode] = "1"
	headers[sofarpc.HeaderCmdType] = "1"
	headers[sofarpc.HeaderCmdCode] = "1"
	headers[sofarpc.HeaderVersion] = "0"
	headers[sofarpc.HeaderCodec] = "11"
	headers[sofarpc.HeaderHeaderLen] = "0"
	headers[sofarpc.HeaderTimeout] = "3000"

	return headers
}
