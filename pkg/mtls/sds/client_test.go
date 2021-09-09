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

	"github.com/stretchr/testify/require"
	"mosn.io/mosn/pkg/types"
)

func testClean() {
	CloseSdsClient()
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
		client := NewSdsClientSingleton(cfg)
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

	t.Run("create an error sds stream client, uprage to success", func(t *testing.T) {
		testClean()
		retryInterval = 100 * time.Millisecond
		SubscriberRetryPeriod = 3 * retryInterval
		client := NewSdsClientSingleton(nil)
		finish := make(chan struct{})
		client.AddUpdateCallback("test", func(name string, secret *types.SdsSecret) {
			close(finish)
		})
		time.Sleep(2 * retryInterval)
		// upragde to connect succes, but send/recv error
		_ = NewSdsClientSingleton(&MockSdsConfig{
			ErrorStr: "test error",
		})
		time.Sleep(2 * retryInterval)
		// upragde to success
		_ = NewSdsClientSingleton(&MockSdsConfig{
			Timeout: 2 * retryInterval, // timeout should greater than retry interval
		})
		select {
		case <-finish:
			testClean()
		case <-time.After(10 * time.Second):
			t.Fatal("sds timeout")
		}
	})

	t.Run("test delete update callback", func(t *testing.T) {
		testClean()
		client := NewSdsClientSingleton(&MockSdsConfig{
			Timeout: time.Second,
		})
		client.AddUpdateCallback("test", func(name string, secret *types.SdsSecret) {
		})
		c := client.(*SdsClientImpl)
		require.Len(t, c.SdsCallbackMap, 1)
		require.Nil(t, client.DeleteUpdateCallback("test"))
		require.Len(t, c.SdsCallbackMap, 0)
	})
}
