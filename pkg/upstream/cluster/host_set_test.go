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

package cluster

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

func newSimpleMockHost(addr string, metaValue string) *mockHost {
	return &mockHost{
		addr: addr,
		meta: api.Metadata{
			"key": metaValue,
		},
	}
}

type simpleMockHostConfig struct {
	addr      string
	metaValue string
}

func TestHostSetDistinct(t *testing.T) {
	hs := &hostSet{}
	ip := "127.0.0.1"
	var hosts []types.Host
	for i := 0; i < 5; i++ {
		host := &mockHost{
			addr: ip,
		}
		hosts = append(hosts, host)
	}
	hs.setFinalHost(hosts)
	if hs.Size() != 1 {
		t.Fatal("hostset distinct failed")
	}
}

func TestHostSet(t *testing.T) {
	var hosts []types.Host
	for i := 1; i < 10; i++ {
		host := &mockHost{
			addr: "127.0.0." + strconv.Itoa(i),
		}
		hosts = append(hosts, host)
	}
	hs := &hostSet{allHosts: hosts}
	require.Equal(t, hs.Size(), len(hosts))
	require.Equal(t, hs.Get(1), hosts[1])
	i := 0
	hs.Range(func(host types.Host) bool {
		i++
		return true
	})
	require.Equal(t, i, 9)
}
