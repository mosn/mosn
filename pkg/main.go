package main

import (
	"time"
	"gitlab.alipay-inc.com/afe/mosn/pkg/xds"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

func main() {
	logPath := "/tmp/xdsclient.log"
	logLevel := log.INFO
	err := log.InitDefaultLogger(logPath, log.LogLevel(logLevel))
	if err != nil {
		return
	}
	callbacks := &xds.Callbacks{}
	go xds.Start(callbacks)

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
