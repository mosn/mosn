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
	"time"

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
)

type HealthCheckLogger struct {
	output string
	logger *log.Logger
}

var (
	// timestamp host health_status current_result status_changed additional info(current_code domain path)
	defaultHealthCheckFormat = "time:%d host:%s health_status:%v current_result:%v status_changed:%v %s"
)

// NewHealthCheckLog
func NewHealthCheckLog(output string) *HealthCheckLogger {
	lg, err := log.GetOrCreateLogger(output, nil)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] new health check log failed, %v", err)
	}

	l := &HealthCheckLogger{
		output: output,
		logger: lg,
	}

	return l
}

func boolToInt(status bool) int {
	if status {
		return 1
	}
	return 0
}

func (l *HealthCheckLogger) LogUpdate(host types.Host, changed bool, isHealthy bool, info string) {
	status := false
	if host.Health() {
		status = true
	}

	l.Log(defaultHealthCheckFormat, time.Now().Unix(), host.AddressString(), boolToInt(status), boolToInt(isHealthy), boolToInt(changed), info)
}

func (l *HealthCheckLogger) Log(format string, args ...interface{}) {
	if l.logger == nil {
		return
	}

	s := fmt.Sprintf(format, args...)
	buf := log.GetLogBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}

	l.logger.Print(buf, true)
}
