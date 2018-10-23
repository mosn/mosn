/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"io/ioutil"
	"sync"

	"github.com/alipay/sofa-mosn/pkg/admin"
	"github.com/alipay/sofa-mosn/pkg/log"
)

var fileMutex = new(sync.Mutex)

func dump(dirty bool) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	if dirty {
		//log.DefaultLogger.Println("dump config to: ", configPath)
		log.DefaultLogger.Debugf("dump config content: %+v", config)

		//update mosn_config
		admin.SetMOSNConfig(config)
		//todo: ignore zero values in config struct @boqin
		content, err := json.MarshalIndent(config, "", "  ")
		if err == nil {
			err = ioutil.WriteFile(configPath, content, 0644)
		}

		if err != nil {
			log.DefaultLogger.Errorf("dump config failed, caused by: " + err.Error())
		}
	} else {
		log.DefaultLogger.Infof("config is clean no needed to dump")
	}
}
