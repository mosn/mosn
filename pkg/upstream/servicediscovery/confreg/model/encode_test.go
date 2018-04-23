package model

import (
	"testing"
	"github.com/golang/protobuf/proto"
	"fmt"
)

func Test_Encode(t *testing.T) {

	publisherRegisterPb := &PublisherRegisterPb{}

	registerPb := BaseRegisterPb{}

	registerPb.AppName = "test"

	registerPb.ClientId = "clientId"
	registerPb.DataId = "dataId"
	registerPb.DataInfoId = "DataInfoId"
	registerPb.EventType = "eventType"
	registerPb.Group = "group"
	registerPb.InstanceId = "InstanceId"
	registerPb.Ip = "ip"
	registerPb.Port = 1200
	registerPb.ProcessId = "ProcessId"
	registerPb.RegistId = "RegistId"
	registerPb.Version = 1
	registerPb.Timestamp = 1
	atr := make(map[string]string, 1)
	atr["a"] = "b"
	registerPb.Attributes = atr
	publisherRegisterPb.BaseRegister = &registerPb

	dataList := make([]*DataBoxPb, 0)
	dataBoxPb := DataBoxPb{"data"}
	dataList = append(dataList, &dataBoxPb)
	publisherRegisterPb.DataList = dataList

	pData, err := proto.Marshal(publisherRegisterPb)
	if err != nil {
		panic(err)
	} else {

		for _,d:=range pData {
		//	fmt.Print(d)
			fmt.Printf("%02x",d)

		}
	}

}
