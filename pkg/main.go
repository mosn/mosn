package main

import (
	"time"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"io/ioutil"
	"encoding/json"

	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
)

var serviceCluster string = "hsf-client-demo"
var serviceNode string = "sidecar~10.151.99.229~hsf-client-demo-v1-0.hsfdemo~hsfdemo.svc.cluster.local"

func main() {
	logPath := "/tmp/xdsclient.log"
	logLevel := log.INFO
	err := log.InitDefaultLogger(logPath, log.LogLevel(logLevel))
	if err != nil {
		return
	}

	content, err := ioutil.ReadFile("/tmp/mosn.json")
	if err != nil {
		return
	}

	var config config.MOSNConfig

	json.Unmarshal(content, &config)

	go xds.Start(&config, serviceCluster, serviceNode)
	xds.WaitForWarmUp()

	t1 := time.NewTimer(time.Second * 10)
	for {
		select {
		case <-t1.C:
			xds.Stop()
			return
		}
	}
	log.CloseAll()
}
