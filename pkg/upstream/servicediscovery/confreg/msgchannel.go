package confreg

import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type  MsgChannel struct{
	ServiceInfo
	MsgChannelCB	MsgChanCallback
}

type MsgChanCallback func([]string)

type ServiceInfo struct{
	ServiceSet []string`json:"service_set"`
}

func (m * MsgChannel)GetSrvInfo(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	body_str := string(body)

	fmt.Println("[MESH HTTP GET]",body_str)

	var msg ServiceInfo
	if err := json.Unmarshal(body,&msg); err == nil {
		m.MsgChannelCB(msg.ServiceSet)
		fmt.Fprint(w,"OK")
	} else {
		fmt.Println(err)
		fmt.Fprint(w,"Error")
	}
}

func (m *MsgChannel) StartChannel(){

	fmt.Println("[MESH HTTP LISTEN]")
	http.HandleFunc("/serviceinfo/", m.GetSrvInfo)
	if err := http.ListenAndServe("localhost:8888", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}