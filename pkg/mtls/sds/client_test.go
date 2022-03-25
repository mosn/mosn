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

package sds

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosn.io/mosn/pkg/types"
)

func testClean() {
	CloseAllSdsClient()
	sdsPostCallback = nil
}

func TestSdsClient(t *testing.T) {
	// Init
	RegisterSdsStreamClientFactory(NewMockSdsStreamClient)

	t.Run("create a new sds client, mock a certificate request registered", func(t *testing.T) {
		testClean()
		cfg := &MockSdsConfig{
			Timeout: time.Second,
		}
		finish := make(chan struct{})
		callbacks := map[string]bool{}
		// Post callback will be called after update callback
		SetSdsPostCallback(func() {
			require.True(t, callbacks["test"])
			close(finish)
		})
		client := NewSdsClient("testIndex", cfg)
		client.AddUpdateCallback("test", func(name string, secret *types.SdsSecret) {
			if name == secret.Name {
				callbacks[name] = true
			}
		})
		select {
		case <-finish:
			testClean()
		case <-time.After(5 * time.Second):
			t.Fatal("sds timeout")
		}
	})

	t.Run("test delete update callback", func(t *testing.T) {
		testClean()
		client := NewSdsClient("testIndex", &MockSdsConfig{
			Timeout: time.Second,
		})
		client.AddUpdateCallback("test", func(name string, secret *types.SdsSecret) {
		})
		c := client.(*SdsClientImpl)
		require.Len(t, c.SdsCallbackMap, 1)
		require.Nil(t, client.DeleteUpdateCallback("test"))
		require.Len(t, c.SdsCallbackMap, 0)
	})

	t.Run("return same client on same index", func(t *testing.T) {
		client1 := NewSdsClient("index1", nil)
		client2 := NewSdsClient("index2", nil)
		assert.NotEqual(t, client1, client2)
		client3 := NewSdsClient("index1", &MockSdsConfig{Timeout: time.Second})
		assert.Equal(t, client1, client3)
	})
}
