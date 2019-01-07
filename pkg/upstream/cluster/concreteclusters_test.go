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
	"fmt"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/types"
)

type mockHost struct {
	types.Host
	ip string
}

func (h *mockHost) AddressString() string {
	return h.ip
}

func (h *mockHost) Weight() uint32 {
	return 0
}

func (h *mockHost) SetWeight(uint32) {
}

type ipPool struct {
	idx int
	ips []string
}

func (pool *ipPool) Get() string {
	ip := pool.ips[pool.idx]
	pool.idx++
	return ip
}

func (pool *ipPool) MakeHosts(size int) []types.Host {
	hosts := make([]types.Host, size)
	for i := 0; i < size; i++ {
		host := &mockHost{
			ip: pool.Get(),
		}
		hosts[i] = host
	}
	return hosts
}

// makePool makes ${size} ips in a ipPool
func makePool(size int) *ipPool {
	var start int64 = 3221291264 // 192.1.1.0
	ips := make([]string, size)
	for i := 0; i < size; i++ {
		ip := start + int64(i)
		ips[i] = fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
	}
	return &ipPool{
		ips: ips,
	}
}

func benchAddHost(b *testing.B, count int) {
	c := &dynamicClusterBase{}
	pool := makePool(2 * count)
	oldHost := pool.MakeHosts(count)
	newHost := pool.MakeHosts(count)
	newHost = append(newHost, oldHost...)
	for i := 0; i < b.N; i++ {
		c.updateDynamicHostList(newHost, oldHost)
	}
}

// add 100 new host
func BenchmarkUpdateDynamicHostList_AddHost100(b *testing.B) {
	benchAddHost(b, 100)
}

// add 500 new host
func BenchmarkUpdateDynamicHostList_AddHost500(b *testing.B) {
	benchAddHost(b, 500)
}

func benchUpdate(b *testing.B, count int) {
	c := &dynamicClusterBase{}
	pool := makePool(3 * count)
	oldHost := pool.MakeHosts(2 * count)
	newHost := pool.MakeHosts(count)
	newHost = append(newHost, oldHost[:count]...)
	for i := 0; i < b.N; i++ {
		c.updateDynamicHostList(newHost, oldHost)
	}
}

// add 50 host, delete 50 host
func BenchmarkUpdateDynamicHostList_UpdateHost50(b *testing.B) {
	benchUpdate(b, 50)
}

// add 500 host ,delete 500 host
func BenchmarkUpdateDynamicHostList_UpdateHost500(b *testing.B) {
	benchUpdate(b, 500)
}

func BenchmarkUpdateDynamicHostList_AddMultipleTimes(b *testing.B) {
	c := &dynamicClusterBase{}
	pool := makePool(45 * 112)
	var final []types.Host
	adds := [][]types.Host{}
	for i := 0; i < 45; i++ {
		hosts := pool.MakeHosts(112)
		adds = append(adds, hosts)
	}
	var m []types.Host
	for i := 0; i < 45; i++ {
		newHosts := append(final, adds[i]...)
		cur := make([]types.Host, len(final))
		copy(cur, final)
		_, final, _, m = c.updateDynamicHostList(newHosts, cur)
		fmt.Println(len(final), len(m))
	}
}

func TestUpdateDynamicHostList(t *testing.T) {
	c := &dynamicClusterBase{}
	pool := makePool(100)
	saveHosts := []types.Host{}
	// test Add
	for i := 1; i < 5; i++ {
		newHosts := append(saveHosts, pool.MakeHosts(10)...)
		curHosts := make([]types.Host, len(saveHosts))
		copy(curHosts, saveHosts)
		changed, final, added, removed := c.updateDynamicHostList(newHosts, curHosts)
		if !(changed && len(final) == i*10 && len(added) == 10 && len(removed) == 0) {
			t.Error("update result unexpected", i, changed, len(final), len(added), len(removed))
		}
		saveHosts = final
	}
	// test Update (add and delete)
	for i := 1; i < 5; i++ {
		newHosts := []types.Host{}
		newHosts = append(newHosts, saveHosts[1:]...)     // delete one
		newHosts = append(newHosts, pool.MakeHosts(5)...) // add five
		curHosts := make([]types.Host, len(saveHosts))
		copy(curHosts, saveHosts)
		changed, final, added, removed := c.updateDynamicHostList(newHosts, curHosts)
		if !(changed && len(final) == 40+i*(4) && len(added) == 5 && len(removed) == 1) {
			t.Error("update result unexpected", i, changed, len(final), len(added), len(removed))
		}
		saveHosts = final
	}

}
