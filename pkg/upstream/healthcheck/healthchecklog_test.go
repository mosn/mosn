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
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHealthCheckLog(t *testing.T) {
	path := "/tmp/hc.log"
	os.Remove(path)
	hc := NewHealthCheckLogger(path)
	if hc != nil {
		return
	}
	host := &mockHost{
		addr: "localhost:5300",
	}
	hc.Log(host, false, false)
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("health check log file open failed, err:%v", err)
	}
	// "time:%d host:%s health_status:%v current_result:%v status_changed:%v %s"
	expected := fmt.Sprintf("time:%d host:%s health_status:%v current_result:%v status_changed:%v",
		time.Now().Unix(), "localhost:5300", 0, 0, 0)
	if !strings.Contains(string(dat), expected) {
		t.Errorf("health check log failed, data:%s, expected:%s", string(dat), expected)
	}
}
