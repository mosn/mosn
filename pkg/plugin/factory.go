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

package plugin

import (
	"sync"
)

var pluginFactories = make(map[string]*Client)
var pluginLock sync.Mutex

// Register called by plugin client and start up the plugin main process.
func Register(name string, config *Config) (*Client, error) {
	pluginLock.Lock()
	defer pluginLock.Unlock()

	if c, ok := pluginFactories[name]; ok {
		return c, nil
	}
	c, err := newClient(name, config)
	if err != nil {
		return nil, err
	}
	pluginFactories[name] = c

	return c, nil
}
