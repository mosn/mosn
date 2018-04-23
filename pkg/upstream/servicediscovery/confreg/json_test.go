package registry

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_PublishRequestJson(t *testing.T) {
	var jsonStr string = `{"providerMetaInfo":{"appName":"testApp","protocol":"bolt","serializeType":"hessian2","version":"4.0"},"serviceName":"aa"}`
	var msg PublishServiceRequest
	if err := json.Unmarshal([]byte(jsonStr), &msg); err == nil {
		//m.MsgChannelCB(body_str)
		fmt.Print(msg.ServiceName)
		fmt.Print(msg)
	} else {
		fmt.Println(err)
		t.Error("error")
	}
}
