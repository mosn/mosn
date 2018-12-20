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

package stats

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"encoding/json"

	"github.com/alipay/sofa-mosn/pkg/types"
)

func addMetrics() {
	// add metrics data
	typs := []string{"typ1", "typ2"}
	namespaces := []string{"ns1", "ns2", "ns3"}
	for _, typ := range typs {
		for _, ns := range namespaces {
			s := NewStats(typ, ns)
			for i := 0; i < 10; i++ {
				s.Counter(fmt.Sprintf("counter.%d", i)).Inc(1)
				s.Gauge(fmt.Sprintf("gauge.%d", i)).Update(1)
			}
			h := s.Histogram("histogram")
			for i := 0; i < 10; i++ {
				h.Update(1)
			}
		}
	}
}

func TestTransferData(t *testing.T) {
	ResetAll()
	addMetrics()
	res1, _ := json.Marshal(GetAllRegistries())
	// get transfer data
	b, err := makesTransferData()
	if err != nil {
		t.Error(err)
		return
	}
	// clear for new
	ResetAll()
	if err := readTransferData(b); err != nil {
		t.Error(err)
		return
	}
	res2, _ := json.Marshal(GetAllRegistries())
	if !reflect.DeepEqual(res1, res2) {
		t.Error("transfer data not matched")
	}

}

func TestTransferWithSocket(t *testing.T) {
	// set env to run TransferServer
	os.Setenv(types.GracefulRestart, "true")
	// set domain socket path
	TransferDomainSocket = "/tmp/stats.sock"
	ResetAll()
	addMetrics()
	res1, _ := json.Marshal(GetAllRegistries())
	ch := make(chan bool)
	go TransferServer(30*time.Second, ch)
	// Wait Server start
	time.Sleep(2 * time.Second)
	defer func() {
		if _, err := os.Stat(TransferDomainSocket); err == nil {
			os.Remove(TransferDomainSocket)
		}
	}()
	body, err := makesTransferData()
	if err != nil {
		t.Error(err)
		return
	}
	ResetAll()
	transferMetrics(body, true, 5*time.Second) // client block, wait server response
	//transferMetrics(body, false, 0)
	//<-ch  // server receive a conn
	res2, _ := json.Marshal(GetAllRegistries())
	if !reflect.DeepEqual(res1, res2) {
		t.Error("transfer data not matched")
	}
}
