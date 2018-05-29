package config

import (
	"encoding/json"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"

	"io/ioutil"
	"sync"
)

var fileMutex *sync.Mutex = new(sync.Mutex)

func Dump(dirty bool) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	if dirty {
		//log.DefaultLogger.Println("dump config to: ", ConfigPath)
		log.DefaultLogger.Debugf("dump config content: %+v", config)

		//todo: ignore zero values in config struct @boqin
		content, err := json.MarshalIndent(config, "", "  ")
		if err == nil {
			err = ioutil.WriteFile(ConfigPath, content, 0644)
		}

		if err != nil {
			log.DefaultLogger.Errorf("dump config failed, caused by: " + err.Error())
		}
	} else {
		log.DefaultLogger.Infof("config is clean no needed to dump")
	}
}