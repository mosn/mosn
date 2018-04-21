package registry

import (
    "fmt"
    "strconv"
)

const PUBLISHER_REGISTER_CLASS = "com.alipay.sofa.registry.common.model.client.pb.PublisherRegisterPb"
const SUBSCRIBER_REGISTER_CLASS = "com.alipay.sofa.registry.common.model.client.pb.SubscriberRegisterPb"

var PUBLISHER_REGISTER_CLASS_LEN = strconv.Itoa(len(PUBLISHER_REGISTER_CLASS))
var SUBSCRIBER_REGISTER_CLASS_LEN = strconv.Itoa(len(SUBSCRIBER_REGISTER_CLASS))

func BuildBoltPublishRequestCommand(contentLen int, reqId string) map[string]string {
    headers := buildBasicRequestCommand()
    headers["x-mosn-sofarpc-headers-property-requestid"] = reqId
    headers["x-mosn-sofarpc-headers-property-classname"] = PUBLISHER_REGISTER_CLASS
    headers["x-mosn-sofarpc-headers-property-classlength"] = PUBLISHER_REGISTER_CLASS_LEN
    headers["x-mosn-sofarpc-headers-property-contentlength"] = fmt.Sprintf("%d", contentLen)

    return headers
}

func BuildBoltSubscribeRequestCommand(contentLen int, reqId string) map[string]string {
    headers := buildBasicRequestCommand()
    headers["x-mosn-sofarpc-headers-property-requestid"] = reqId
    headers["x-mosn-sofarpc-headers-property-classname"] = SUBSCRIBER_REGISTER_CLASS
    headers["x-mosn-sofarpc-headers-property-classlength"] = SUBSCRIBER_REGISTER_CLASS_LEN
    headers["x-mosn-sofarpc-headers-property-contentlength"] = fmt.Sprintf("%d", contentLen)

    return headers
}

func buildBasicRequestCommand() map[string]string {
    headers := make(map[string]string)
    headers["x-mosn-sofarpc-headers-property-protocol"] = "1"
    headers["x-mosn-sofarpc-headers-property-cmdtype"] = "1"
    headers["x-mosn-sofarpc-headers-property-cmdcode"] = "1"
    headers["x-mosn-sofarpc-headers-property-version"] = "0"
    //headers["x-mosn-sofarpc-headers-property-requestid"] = "114"
    headers["x-mosn-sofarpc-headers-property-codec"] = "11"
    headers["x-mosn-sofarpc-headers-property-headerlength"] = "0"
    headers["x-mosn-sofarpc-headers-property-timeout"] = "3000"

    return headers
}
