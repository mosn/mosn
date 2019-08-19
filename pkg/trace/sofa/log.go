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
	"sofastack.io/sofa-mosn/pkg/types"
	"errors"
)

type tracelog struct {
	ingressLogger *log.Logger
	egressLogger  *log.Logger
	init          sync.Once
}

var (
	logMap           = make(map[types.Protocol]*tracelog)
	ErrIngressLogger = errors.New("ingress logger cannot be empty")
	ErrEgressLogger  = errors.New("egress logger cannot be empty")
)

func Init(protocol types.Protocol, logRoot, logIngress, logEgress string) (err error) {
	tl, ok := logMap[protocol]
	if !ok {
		tl = &tracelog{}
		logMap[protocol] = tl
	}

	tl.init.Do(func() {
		if logRoot == "" {
			// get default log root
			usr, err := user.Current()
			if err != nil {
				return
			}
			logRoot = usr.HomeDir + "/logs/tracelog/mosn/"
		}

		if logIngress == "" {
			err = ErrIngressLogger
			return
		}

		tl.ingressLogger, err = log.GetOrCreateLogger(logRoot + logIngress, nil)
		if err != nil {
			return
		}

		if logEgress == "" {
			err = ErrEgressLogger
			return
		}

		tl.egressLogger, err = log.GetOrCreateLogger(logRoot + logEgress, nil)
		if err != nil {
			return
		}
	})
	return
}

func GetIngressLogger(protocol types.Protocol) *log.Logger {
	return logMap[protocol].ingressLogger
}

func GetEgressLogger(protocol types.Protocol) *log.Logger {
	return logMap[protocol].egressLogger
}
