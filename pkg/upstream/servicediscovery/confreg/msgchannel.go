package confreg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type MsgChannel struct {
	ServiceInfo
	MsgChannelCB MsgChanCallback
}

type MsgChanCallback func([]string)

type ServiceInfo struct {
	ServiceSet []string `json:"service_set"`
}

func init(){
	//MsgChannel Server
	var MsgChan = MsgChannel{}
	go func(){
		MsgChan.StartChannel()
	}()
}

//发布服务
func (m *MsgChannel) PublishService(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		log.DefaultLogger.Errorf("[MSG CHANNEL] expect method 'POST' but got '%s'", r.Method)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	body_str := string(body)

	log.DefaultLogger.Infof("[MSG CHANNEL] MESH HTTP GET :%s", body_str)

	var msg PublishServiceRequest
	if err := json.Unmarshal(body, &msg); err == nil {
		//m.MsgChannelCB(body_str)

		// 这里去配置中心发布.同步返回结果

		var result = PublishServiceResult{}
		result.Success = true

		m, err := json.Marshal(result)
		if err == nil {
			w.Write(m)
		} else {
			log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		}
	} else {
		log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		fmt.Fprint(w, "Error")
	}
}

//取消发布服务
func (m *MsgChannel) UnPublishService(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		log.DefaultLogger.Errorf("[MSG CHANNEL] expect method 'POST' but got '%s'", r.Method)
	}

	body, _ := ioutil.ReadAll(r.Body)
	body_str := string(body)

	log.DefaultLogger.Infof("[MSG CHANNEL] MESH HTTP GET :%s", body_str)

	var msg UnPublishServiceRequest
	if err := json.Unmarshal(body, &msg); err == nil {
		//m.MsgChannelCB(body_str)

		// 这里去配置中心取消发布.同步返回结果

		var result = UnPublishServiceResult{}
		result.Success = true
		m, err := json.Marshal(result)
		if err == nil {
			w.Write(m)
		} else {
			log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		}
	} else {
		log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		fmt.Fprint(w, "Error")
	}
}

// 订阅服务

func (m *MsgChannel) SubscribeService(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		log.DefaultLogger.Errorf("[MSG CHANNEL] expect method 'POST' but got '%s'", r.Method)
	}

	body, _ := ioutil.ReadAll(r.Body)
	body_str := string(body)

	log.DefaultLogger.Infof("[MSG CHANNEL] MESH HTTP GET :%s", body_str)

	var msg SubscribeServiceRequest
	if err := json.Unmarshal(body, &msg); err == nil {
		//m.MsgChannelCB(body_str)

		// 这里实现订阅服务,并返回结果,这里返回的 datas 中只需要从返回的所有列表中挑几个就行了.
		// 不用全部返回,因为客户端拿到这个信息,主要是获取其中的序列化,编码等信息

		var result = SubscribeServiceResult{}
		result.ServiceName = "xxx"
		result.Success = true
		result.Datas = []string{"1", "2"}
		m, err := json.Marshal(result)
		if err == nil {
			w.Write(m)
		} else {
			log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		}
	} else {
		log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		fmt.Fprint(w, "Error")
	}
}

// 取消订阅服务

func (m *MsgChannel) UnSubscribeService(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		log.DefaultLogger.Errorf("[MSG CHANNEL] expect method 'POST' but got '%s'", r.Method)
	}

	body, _ := ioutil.ReadAll(r.Body)
	body_str := string(body)

	log.DefaultLogger.Infof("[MSG CHANNEL] MESH HTTP GET :%s", body_str)

	var msg UnSubscribeServiceRequest
	if err := json.Unmarshal(body, &msg); err == nil {
		//m.MsgChannelCB(body_str)

		// 这里去配置中心取消订阅.同步返回结果

		var result = UnSubscribeServiceResult{}
		result.Success = true
		m, err := json.Marshal(result)
		if err == nil {
			w.Write(m)
		} else {
			log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		}
	} else {
		log.DefaultLogger.Errorf("[MSG CHANNEL] %s", err)
		fmt.Fprint(w, "Error")
	}
}

func (m *MsgChannel) StartChannel() {

	fmt.Println("[MESH HTTP LISTEN]")
	http.HandleFunc("/publishService", m.PublishService)
	http.HandleFunc("/unPublishService", m.UnPublishService)
	http.HandleFunc("/subscribeService", m.SubscribeService)
	http.HandleFunc("/unSubscribeService", m.UnSubscribeService)

	if err := http.ListenAndServe("0.0.0.0:8888", nil); err != nil {
		log.DefaultLogger.Errorf("[MSG CHANNEL] ListenAndServe: %s", err)
	}
}
