package main

import (
	"time"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"io/ioutil"
	"encoding/json"

	"gitlab.alipay-inc.com/afe/mosn/pkg/config"
)

var serviceCluster string = "s1"
var serviceNode string = "sidecar~10.151.99.219~s1-v1-794c4588fb-ghlc9.hw~hw.svc.cluster.local"

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

	t1 := time.NewTimer(time.Second * 60)
	for {
		select {
		case <-t1.C:
			xds.Stop()
			return
		}
	}
	log.CloseAll()
}
