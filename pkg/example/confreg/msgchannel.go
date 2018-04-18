//An example of using http json msg channel
package main


import (
	"fmt"
	"io/ioutil"
	"net/http"
	"bytes"
	"encoding/json"
	"gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg"
	"time"
	"sync"
)

var msg = confreg.ServiceInfo{
	[]string{"service1","service2","service3","service4"},
}

var serInfo []string

func GetSerivceInfo(s []string){

	//Add other logic
	fmt.Println("[CALL BACK GET]",s)
	return
}

var msgChan = confreg.MsgChannel{
	msg,
	GetSerivceInfo,
}

func httpreceive(){

	m,_ := json.Marshal(msg)
	req_new := bytes.NewBuffer([]byte(m))
	request, _ := http.NewRequest("POST", "http://localhost:8888/serviceinfo/", req_new)
	request.Header.Set("Content-type", "application/json")
	client := &http.Client{}
	response, _ := client.Do(request)

	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		fmt.Println("[SOFA CLIENT RECEIVE]",string(body))
	}
}

func main() {

	var wg sync.WaitGroup
	channelReadyChan := make(chan bool)
	wg.Add(2)

	go func(){
		go msgChan.StartChannel("localhost:8888")
		channelReadyChan <- true
	}()

	select {
	case <-time.After(2 * time.Second):
	}

	go func(){
		select{
		case <- channelReadyChan:
			go httpreceive()
		}
	}()

	wg.Wait()
}

