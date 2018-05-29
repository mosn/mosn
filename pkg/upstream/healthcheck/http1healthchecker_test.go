package healthcheck

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	IntervalDur time.Duration = 1 * time.Second
	TimeoutDur  time.Duration = 10 * time.Second
	//APPCheckPointURL string        = "http://127.0.0.1:9500/checkService"
	APPCheckPointURL = "http://instpay.rz00b.alipay.net:9500/checkService"
)

func init(){
	log.InitDefaultLogger("",log.DEBUG)
}

func TestHttp1HealthCheck_Start(t *testing.T) {
	StartHttpHealthCheck(IntervalDur, TimeoutDur, APPCheckPointURL, onAppInterval, onTimeout)
	log.DefaultLogger.Debugf("wait 15 seconds")
	
	time.Sleep(3600 * time.Second)
}

func onAppInterval(path string, hcResetTimeOut func()) {
	log.DefaultLogger.Debugf("Send Http Get %s",path)
	resp, err := http.Get(path)
	if err != nil {
		// handle error
		log.DefaultLogger.Debugf(" Get Error: %s from path %s",
			err.Error(),path)
		// wait next tick
	}
	
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	
	if err != nil {
		// handle error
		log.DefaultLogger.Debugf(" Read %s %s", path,
			err.Error())
		// wait next tick
	}
	
	res := string(body)
	//get first line
	idx := strings.Index(res, "\n")
	
	if idx == -1 {
	}
	
	if res[:idx] == "passed:true" {
		//shc.handleSuccess()
		log.DefaultLogger.Debugf("APP Health Checks got success")
		hcResetTimeOut()
	} else {
		// wait next tick
	}
}

func onTimeout() {
	log.DefaultLogger.Debugf("Timeout")
}
