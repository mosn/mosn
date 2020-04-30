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

package configmanager

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/utils"
)

var (
	once    sync.Once
	lock    sync.Mutex
	dumping int32
)

func DumpLock() {
	lock.Lock()
}

func DumpUnlock() {
	lock.Unlock()
}

func setDump() {
	atomic.CompareAndSwapInt32(&dumping, 0, 1)
}

func getDump() bool {
	return atomic.CompareAndSwapInt32(&dumping, 1, 0)
}

func dump(dirty bool) {
	if dirty {
		setDump()
	}
}

func DumpConfig() {
	if getDump() {
		log.DefaultLogger.Debugf("[config] [dump] dump config content: %+v", config)

		// use golang original json lib, so the marshal ident can handle MarshalJSON interface implement correctly
		configLock.Lock()
		defer configLock.Unlock()
		content, err := json.MarshalIndent(config, "", "  ")
		if err == nil {
			err = utils.WriteFileSafety(configPath, content, 0644)
		}

		if err != nil {
			log.DefaultLogger.Alertf(types.ErrorKeyConfigDump, "dump config failed, caused by: "+err.Error())
			// add retry if dump failed
			setDump()
		}
	}
}

// DumpConfigHandler should be called in a goroutine
// we call it in mosn/starter with GoWithRecover, which can handle the panic information
func DumpConfigHandler() {
	once.Do(func() {
		for {
			time.Sleep(3 * time.Second)

			DumpLock()
			DumpConfig()
			DumpUnlock()
		}
	})
}
