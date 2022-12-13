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
package healthcheck

import (
	"fmt"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
	"time"
)

type hcLogCreator func(path string) types.HealthCheckLog

var gHealthCheckLogCreator hcLogCreator

func SetHealthCheckLogger(creator hcLogCreator)  {
	gHealthCheckLogCreator = creator
}

// NewHealthCheckLogger returns a hc logger
func NewHealthCheckLogger(output string) types.HealthCheckLog {
	if gHealthCheckLogCreator != nil {
		return gHealthCheckLogCreator(output)
	}
	return defaultHealthCheckLogCreator(output)
}

func defaultHealthCheckLogCreator(output string) types.HealthCheckLog {
	if output == "" {
		return nil
	}
	lg, err := log.GetOrCreateLogger(output, nil)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] new health check log failed, %v", err)
	}

	l := &defaultHealthCheckLogger{
		output: output,
		logger: lg,
	}

	return l
}

type defaultHealthCheckLogger struct {
	output string
	logger *log.Logger
}

const (
	// timestamp host health_status current_result status_changed additional info(current_code domain path)
	defaultHealthCheckFormat = "time:%d host:%s health_status:%v current_result:%v status_changed:%v"
)

func (l *defaultHealthCheckLogger) Log(host types.Host, current_status, changed bool) {
	if l.logger == nil {
		return
	}

	s := fmt.Sprintf(defaultHealthCheckFormat, time.Now().Unix(), host.AddressString(), boolToInt(host.Health()),
		boolToInt(current_status), boolToInt(changed))

	buf := log.GetLogBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}

	l.logger.Print(buf, true)
}

func boolToInt(status bool) int {
	if status {
		return 1
	}
	return 0
}