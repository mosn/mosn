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

package sofa

import (
	"sofastack.io/sofa-mosn/pkg/log"
	"sync"
	"os/user"
)

var (
	ingressLogger *log.Logger
	egressLogger  *log.Logger
	logInit       sync.Once
)

func Init(logRoot, logIngress, logEgress string) (err error) {
	logInit.Do(func() {
		if logRoot == "" {
			// get default log root
			usr, err := user.Current()
			if err != nil {
				return
			}
			logRoot = usr.HomeDir + "/logs/tracelog/mosn/"
		}

		if logIngress == "" {
			// get default ingress log path
			logIngress = "rpc-server-digest.log"
		}

		ingressLogger, err = log.GetOrCreateLogger(logRoot + logIngress)
		if err != nil {
			return
		}

		if logEgress == "" {
			// get default egress log path
			logEgress = "rpc-client-digest.log"
		}

		egressLogger, err = log.GetOrCreateLogger(logRoot + logEgress)
		if err != nil {
			return
		}
	})
	return
}

func GetIngressLogger() *log.Logger {
	return ingressLogger
}

func GetEgressLogger() *log.Logger {
	return egressLogger
}
