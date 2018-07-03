package servermanager

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"time"
)

func beforeTest() {
	log.InitDefaultLogger("", log.INFO)
}

func blockThread() {
	for {
		time.Sleep(5 * time.Second)
	}
}
